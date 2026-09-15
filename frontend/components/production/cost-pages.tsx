'use client';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { request, quantity, money } from './execution-api';
import { varianceTone, type OrderCost, type PeriodCost } from './cost-api';
import {
  Badge,
  Empty,
  ErrorMessage,
  Metrics,
  Panel,
  Tabs,
  Workspace,
  date,
  dateTime,
} from './execution-ui';

const message = (error: unknown) => (error instanceof Error ? error.message : 'Unable to load production costs.');
const sum = (values: (string | number)[]) => values.reduce((carry: number, value) => carry + Number(value ?? 0), 0);

function Variance({ value, percent }: { value: string; percent?: string }) {
  return (
    <span className={`execution-badge status-${varianceTone(value)}`}>
      {Number(value) > 0 ? '+' : ''}
      {percent === undefined ? quantity(value) : `${quantity(percent)}%`}
    </span>
  );
}

export function CostIndex() {
  const [items, setItems] = useState<OrderCost[]>([]);
  const [periods, setPeriods] = useState<PeriodCost[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [period, setPeriod] = useState('');
  const [tab, setTab] = useState('Per order');

  useEffect(() => {
    let alive = true;
    Promise.all([
      request<{ items: OrderCost[] }>('/production-costs'),
      request<{ items: PeriodCost[] }>('/production-costs/periods'),
    ])
      .then(([orders, months]) => {
        if (!alive) return;
        setItems(orders.items ?? []);
        setPeriods(months.items ?? []);
      })
      .catch((cause) => { if (alive) setError(message(cause)); })
      .finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const filtered = useMemo(() => items.filter((order) => {
    const matches = `${order.orderNumber} ${order.partNumber} ${order.partName} ${order.plantName}`.toLowerCase().includes(search.toLowerCase());
    return matches && (!period || order.periodStart.slice(0, 7) === period);
  }), [items, search, period]);
  const currency = filtered[0]?.currency || 'IDR';
  const totalMoney = (field: 'totalActual' | 'wipValue' | 'finishedCost') => money(sum(filtered.map((order) => order[field])), currency);
  const produced = sum(filtered.map((order) => order.goodQty));

  const overruns = filtered.filter((order) => Number(order.variance) > 0 && Number(order.goodQty) > 0);

  return (
    <Workspace
      title="Production Cost"
      subtitle="Actual material and process cost per order, taken from the prices frozen on every transaction."
      crumbs={[{ label: 'Production cost' }]}
    >
      <Metrics
        items={[
          { label: 'Actual cost booked', value: totalMoney('totalActual'), hint: `${filtered.length} orders`, tone: 'accent' },
          { label: 'Cost in work in progress', value: totalMoney('wipValue'), hint: 'Not yet finished', tone: 'amber' },
          { label: 'Finished output cost', value: totalMoney('finishedCost'), hint: `${quantity(produced)} good pieces` },
          { label: 'Orders over estimate', value: overruns.length, hint: 'Actual above release estimate', tone: overruns.length ? 'red' : 'green' },
        ]}
      />
      <ErrorMessage error={error} />
      <Panel title="Cost register" note="Finished cost is everything booked on an order less the cost still held in WIP.">
        <Tabs tabs={['Per order', 'Per period']} active={tab} onSelect={setTab} label="Cost views" />
        {tab === 'Per order' ? (
          <>
            <div className="execution-filters">
              <label>
                Search orders
                <input
                  type="search"
                  placeholder="Order, part or plant"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                />
              </label>
              <label>
                Period
                <select aria-label="Period filter" value={period} onChange={(event) => setPeriod(event.target.value)}>
                  <option value="">All periods</option>
                  {[...new Set(periods.map((month) => month.period))].map((month) => (
                    <option key={month} value={month}>{month}</option>
                  ))}
                </select>
              </label>
            </div>
            {loading ? (
              <p className="execution-empty" role="status">Loading production costs…</p>
            ) : filtered.length === 0 ? (
              <Empty title={error ? 'Costs unavailable' : 'No production costs yet'} hint="Costs appear once daily production is recorded." />
            ) : (
              <div className="execution-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Production order</th>
                      <th>Part</th>
                      <th className="numeric">Good qty</th>
                      <th className="numeric">Material</th>
                      <th className="numeric">Process</th>
                      <th className="numeric">Total actual</th>
                      <th className="numeric">In WIP</th>
                      <th className="numeric">Cost / piece</th>
                      <th className="numeric">Estimate / piece</th>
                      <th>Variance</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((order) => (
                      <tr key={order.orderId}>
                        <td>
                          <a href={`/production-costs/${order.orderId}`}>{order.orderNumber}</a>
                          <small>{order.plantName}</small>
                        </td>
                        <td>
                          {order.partNumber}
                          <small>{order.partName}</small>
                        </td>
                        <td className="numeric">
                          {quantity(order.goodQty)}
                          <small>{order.unitCode}</small>
                        </td>
                        <td className="numeric">{money(order.materialActual, order.currency)}</td>
                        <td className="numeric">{money(order.processActual, order.currency)}</td>
                        <td className="numeric"><strong>{money(order.totalActual, order.currency)}</strong></td>
                        <td className="numeric">{money(order.wipValue, order.currency)}</td>
                        <td className="numeric">{money(order.actualUnitCost, order.currency)}</td>
                        <td className="numeric">{money(order.plannedUnitCost, order.currency)}</td>
                        <td>
                          {Number(order.goodQty) > 0 ? <Variance value={order.variance} percent={order.variancePercent} /> : <span className="muted">—</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        ) : periods.length === 0 ? (
          <Empty title="No closed or open periods yet" hint="A period appears once production is recorded in it." />
        ) : (
          <div className="execution-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Period</th>
                  <th>State</th>
                  <th className="numeric">Orders</th>
                  <th className="numeric">Entries</th>
                  <th className="numeric">Processed</th>
                  <th className="numeric">Good</th>
                  <th className="numeric">Reject</th>
                  <th className="numeric">Material</th>
                  <th className="numeric">Process</th>
                  <th className="numeric">Total actual</th>
                </tr>
              </thead>
              <tbody>
                {periods.map((month) => (
                  <tr key={`${month.period}-${month.currency}`}>
                    <td><strong>{month.period}</strong><small>{month.currency}</small></td>
                    <td>
                      <span className={`execution-badge status-${month.closed ? 'completed' : 'in_progress'}`}>
                        {month.closed ? 'CLOSED' : 'OPEN'}
                      </span>
                      {month.closedBy && <small>{month.closedBy}</small>}
                    </td>
                    <td className="numeric">{month.orders}</td>
                    <td className="numeric">{month.entries}</td>
                    <td className="numeric">{quantity(month.processedQty)}</td>
                    <td className="numeric">{quantity(month.goodQty)}</td>
                    <td className="numeric">{quantity(month.rejectQty)}</td>
                    <td className="numeric">{money(month.materialActual, month.currency)}</td>
                    <td className="numeric">{money(month.processActual, month.currency)}</td>
                    <td className="numeric"><strong>{money(month.totalActual, month.currency)}</strong></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
    </Workspace>
  );
}

export function CostDetail({ id }: { id: string }) {
  const [order, setOrder] = useState<OrderCost>();
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      setOrder(await request<OrderCost>(`/production-costs/${id}`));
    } catch (cause) {
      setError(message(cause));
    }
  }, [id]);
  useEffect(() => { void load(); }, [load]);

  return (
    <Workspace
      title={order ? `Cost · ${order.orderNumber}` : 'Production Cost'}
      subtitle={order ? `${order.partNumber} · ${order.partName} · ${order.plantName}` : 'Loading cost breakdown…'}
      crumbs={[{ label: 'Production cost', href: '/production-costs' }, { label: 'Order cost' }]}
      action={
        <>
          <a className="ex-button" href="/production-costs">All costs</a>
          {order && <a className="ex-button" href={`/production-orders/${order.orderId}`}>Production order</a>}
          {order && <a className="ex-button" href={`/production-wip/${order.orderId}`}>WIP ledger</a>}
        </>
      }
    >
      <ErrorMessage error={error} />
      {!order ? (
        !error && <p className="execution-empty" role="status">Loading cost breakdown…</p>
      ) : (
        <>
          <Metrics
            items={[
              { label: 'Finished output cost', value: money(order.finishedCost, order.currency), hint: `${quantity(order.goodQty)} ${order.unitCode} good`, tone: 'accent' },
              { label: 'Cost per finished piece', value: money(order.actualUnitCost, order.currency), hint: `Estimate ${money(order.plannedUnitCost, order.currency)}` },
              { label: 'Cost still in WIP', value: money(order.wipValue, order.currency), hint: 'Between operations', tone: 'amber' },
              {
                label: 'Variance per piece',
                value: money(order.variance, order.currency),
                hint: `${quantity(order.variancePercent)}% against estimate`,
                tone: Number(order.variance) > 0 ? 'red' : 'green',
              },
            ]}
          />

          <div className="execution-snapshot">
            <span><small>Production order</small><a href={`/production-orders/${order.orderId}`}>{order.orderNumber}</a></span>
            <span><small>Planning</small><strong>{order.planNumber}</strong></span>
            <span><small>Status</small><Badge status={order.status} /></span>
            <span><small>Order period</small><strong>{date(order.periodStart)} → {date(order.dueDate)}</strong></span>
            <span><small>Daily entries</small><strong>{order.entryCount}</strong></span>
            <span><small>Last change</small><strong>{dateTime(order.updatedAt)}</strong></span>
          </div>

          <Panel title="Cost build-up" note="Material plus every process step, less the cost still held in work in progress.">
            <div className="execution-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Component</th>
                    <th className="numeric">Estimate at release</th>
                    <th className="numeric">Actual</th>
                    <th className="numeric">Difference</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <strong>Material</strong>
                      <small>Usage × price frozen per entry</small>
                    </td>
                    <td className="numeric">{money(order.materialEstimate, order.currency)}</td>
                    <td className="numeric">{money(order.materialActual, order.currency)}</td>
                    <td className="numeric">{money(Number(order.materialActual) - Number(order.materialEstimate), order.currency)}</td>
                  </tr>
                  {order.operations.map((operation) => (
                    <tr key={operation.operationId}>
                      <td>
                        <strong>{operation.code}</strong>
                        <small>{quantity(operation.processedQty)} processed × {money(operation.rateSnapshot, order.currency)} per piece</small>
                      </td>
                      <td className="numeric">{money(operation.estimateCost, order.currency)}</td>
                      <td className="numeric">{money(operation.actualCost, order.currency)}</td>
                      <td className="numeric">{money(Number(operation.actualCost) - Number(operation.estimateCost), order.currency)}</td>
                    </tr>
                  ))}
                  <tr>
                    <td><strong>Less: cost held in WIP</strong></td>
                    <td className="numeric">—</td>
                    <td className="numeric">{money(-Number(order.wipValue), order.currency)}</td>
                    <td className="numeric">—</td>
                  </tr>
                </tbody>
                <tfoot>
                  <tr>
                    <td>Finished output cost</td>
                    <td className="numeric">{money(order.totalEstimate, order.currency)}</td>
                    <td className="numeric">{money(order.finishedCost, order.currency)}</td>
                    <td className="numeric">{money(Number(order.finishedCost) - Number(order.totalEstimate), order.currency)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </Panel>

          <Panel title="Process cost per operation" note="Each operation books the rate captured when its entries were posted.">
            {order.operations.length === 0 ? (
              <Empty title="No operations recorded" />
            ) : (
              <div className="execution-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th className="numeric">Seq</th>
                      <th>Operation</th>
                      <th className="numeric">Processed</th>
                      <th className="numeric">Good</th>
                      <th className="numeric">Reject</th>
                      <th className="numeric">Rate</th>
                      <th className="numeric">Actual cost</th>
                      <th className="numeric">Cost per good piece</th>
                      <th className="numeric">Entries</th>
                    </tr>
                  </thead>
                  <tbody>
                    {order.operations.map((operation) => (
                      <tr key={operation.operationId}>
                        <td className="numeric">{operation.sequence}</td>
                        <td>
                          <strong>{operation.code}</strong>
                          <small>{operation.final ? 'Final operation · finished output' : operation.name}</small>
                        </td>
                        <td className="numeric">{quantity(operation.processedQty)}</td>
                        <td className="numeric">{quantity(operation.goodQty)}</td>
                        <td className="numeric">{quantity(operation.rejectQty)}</td>
                        <td className="numeric">{money(operation.rateSnapshot, order.currency)}</td>
                        <td className="numeric">{money(operation.actualCost, order.currency)}</td>
                        <td className="numeric">{money(operation.costPerPiece, order.currency)}</td>
                        <td className="numeric">{operation.entries}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Panel>

          <Panel title="Material usage" note="Actual usage against the BOM expectation for the quantity started.">
            {order.materials.length === 0 ? (
              <Empty title="No material recorded" hint="Material is booked at the first operation." />
            ) : (
              <div className="execution-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Material</th>
                      <th className="numeric">Standard usage</th>
                      <th className="numeric">Actual usage</th>
                      <th>Usage variance</th>
                      <th className="numeric">Average price</th>
                      <th className="numeric">Actual cost</th>
                    </tr>
                  </thead>
                  <tbody>
                    {order.materials.map((material) => (
                      <tr key={material.materialId}>
                        <td>
                          <strong>{material.partNumber}</strong>
                          <small>{material.partName}</small>
                        </td>
                        <td className="numeric">{quantity(material.standardQty)}</td>
                        <td className="numeric">
                          {quantity(material.usedQty)}
                          <small>{material.unitCode}</small>
                        </td>
                        <td><Variance value={material.qtyVariance} /></td>
                        <td className="numeric">{money(material.averagePrice, order.currency)}</td>
                        <td className="numeric">{money(material.actualCost, order.currency)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td colSpan={5}>Total material cost</td>
                      <td className="numeric">{money(order.materialActual, order.currency)}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            )}
          </Panel>
        </>
      )}
    </Workspace>
  );
}
