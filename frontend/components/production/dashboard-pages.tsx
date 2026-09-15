'use client';
import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';
import { request, quantity, money } from './execution-api';
import { rejectTone, type ProductionDashboard } from './dashboard-api';
import { Empty, ErrorMessage, Metrics, Panel, Workspace, dateTime } from './execution-ui';

const message = (error: unknown) => (error instanceof Error ? error.message : 'Unable to load the production dashboard.');
const share = (value: string | number, scale: number) => (scale > 0 ? Math.min(100, (Number(value) / scale) * 100) : 0);

/** One labelled bar: a track, a single-hue fill and its value read directly. */
function Bar({ label, title, fill, value, planned }: {
  label: string;
  title: string;
  fill: number;
  value: ReactNode;
  planned?: boolean;
}) {
  return (
    <div className="pd-row">
      <span title={label}>{label}</span>
      <div className={`pd-track${planned ? ' planned' : ''}`} role="img" aria-label={title} title={title}>
        <div className="pd-fill" style={{ width: `${Math.max(0, Math.min(100, fill))}%` }} />
      </div>
      <span className="pd-value">{value}</span>
    </div>
  );
}

export function ProductionDashboardView() {
  const user = useCurrentUser();
  const [data, setData] = useState<ProductionDashboard>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');

  const load = useCallback(async (start?: string, end?: string) => {
    setLoading(true);
    setError('');
    const query = new URLSearchParams();
    if (start) query.set('from', start);
    if (end) query.set('to', end);
    try {
      const result = await request<ProductionDashboard>(`/production-dashboard${query.size ? `?${query}` : ''}`);
      setData(result);
      setFrom(result.filter.from);
      setTo(result.filter.to);
    } catch (cause) {
      setError(message(cause));
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { void load(); }, [load]);

  const costs = Boolean(user?.permissions.includes('production.report'));
  const currency = data?.totals.currency || 'IDR';
  const totalMoney = (field: 'wipValue' | 'totalActual') => money(data?.totals[field] ?? 0, currency);
  const exportQuery = `?from=${data?.filter.from ?? ''}&to=${data?.filter.to ?? ''}`;
  const plannedScale = Math.max(...(data?.parts ?? []).map((part) => Number(part.plannedQty) || 0), 1);
  const wipScale = Math.max(...(data?.processes ?? []).map((process) => Number(process.quantity) || 0), 1);
  const rejectScale = Math.max(...(data?.quality ?? []).map((operation) => Number(operation.rejectRate) || 0), 5);

  return (
    <Workspace
      title="Production Dashboard"
      subtitle="Plan against actual output, work in progress per process, reject rate and cost per part number."
      crumbs={[{ label: 'Dashboard' }]}
      action={
        costs && (
          <>
            <a className="ex-button" href={`/api/production-reports/dashboard.xlsx${exportQuery}`}>Export Excel</a>
            <a className="ex-button" href={`/api/production-reports/dashboard.pdf${exportQuery}`}>Export PDF</a>
          </>
        )
      }
    >
      <Panel title="Reporting period" note="Quantities and cost cover this window; work in progress is always the live balance.">
        <div className="execution-panel-body pd-filters">
          <label>
            From
            <input aria-label="From date" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </label>
          <label>
            To
            <input aria-label="To date" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
          </label>
          <button type="button" className="ex-button primary" disabled={loading} onClick={() => void load(from, to)}>
            Apply period
          </button>
          {data && <span className="execution-footer-note">Generated {dateTime(data.generatedAt)}</span>}
        </div>
      </Panel>

      <ErrorMessage error={error} />

      {loading && !data ? (
        <p className="execution-empty" role="status">Loading production dashboard…</p>
      ) : !data ? null : (
        <>
          <Metrics
            items={[
              { label: 'Planned quantity', value: quantity(data.totals.plannedQty), hint: `${data.totals.activeOrders} orders in period` },
              { label: 'Actual good', value: quantity(data.totals.goodQty), hint: `${quantity(data.totals.achievement)}% of plan`, tone: 'accent' },
              { label: 'Reject rate', value: `${quantity(data.totals.rejectRate)}%`, hint: `${quantity(data.totals.rejectQty)} rejected`, tone: rejectTone(data.totals.rejectRate) },
              { label: 'Work in progress', value: quantity(data.totals.wipQty), hint: costs ? totalMoney('wipValue') : 'Live balance', tone: 'amber' },
              ...(costs ? [{ label: 'Actual cost', value: totalMoney('totalActual'), hint: `${data.totals.entries} daily entries` } as const] : []),
            ]}
          />

          <div className="pd-grid">
            <Panel title="Plan vs actual" note="Good output of the final operation against the released plan target.">
              {data.parts.length === 0 ? (
                <Empty title="No orders in this period" hint="Release a production order to see plan attainment." />
              ) : (
                <>
                  <div className="pd-chart">
                    {data.parts.map((part) => (
                      <Bar
                        key={part.partNumber}
                        label={part.partNumber}
                        planned
                        title={`${part.partNumber}: ${quantity(part.goodQty)} good of ${quantity(part.plannedQty)} planned ${part.unitCode}`}
                        fill={share(part.goodQty, Number(part.plannedQty) || plannedScale)}
                        value={
                          <>
                            {quantity(part.goodQty)} / {quantity(part.plannedQty)}
                            <small>{quantity(part.achievement)}% of plan</small>
                          </>
                        }
                      />
                    ))}
                  </div>
                  <div className="pd-legend">
                    <span><i /> Actual good</span>
                    <span><i className="planned" /> Planned target</span>
                  </div>
                </>
              )}
            </Panel>

            <Panel title="WIP per process" note="Live ledger balance waiting at or staged for each operation.">
              {data.processes.length === 0 ? (
                <Empty title="No work in progress" hint="Balances appear once an operation reports good output." />
              ) : (
                <div className="pd-chart">
                  {data.processes.map((process) => (
                    <Bar
                      key={process.code}
                      label={process.code}
                      title={`${process.code}: ${quantity(process.quantity)} in progress`}
                      fill={share(process.quantity, wipScale)}
                      value={
                        <>
                          {quantity(process.quantity)}
                          {costs && <small>{money(process.value, currency)}</small>}
                        </>
                      }
                    />
                  ))}
                </div>
              )}
            </Panel>

            <Panel title="Reject rate per operation" note="Rejected pieces against everything processed at that operation.">
              {data.quality.length === 0 ? (
                <Empty title="No production recorded" hint="Reject rates follow daily production entries." />
              ) : (
                <div className="pd-chart">
                  {data.quality.map((operation) => (
                    <Bar
                      key={operation.code}
                      label={operation.code}
                      title={`${operation.code}: ${quantity(operation.rejectRate)}% reject rate of ${quantity(operation.processedQty)} processed`}
                      fill={share(operation.rejectRate, rejectScale)}
                      value={
                        <>
                          {quantity(operation.rejectRate)}%
                          <small>{quantity(operation.rejectQty)} of {quantity(operation.processedQty)}</small>
                        </>
                      }
                    />
                  ))}
                </div>
              )}
            </Panel>

            <Panel title="Output per part" note="The same figures as the plan chart, read as numbers.">
              {data.parts.length === 0 ? (
                <Empty title="No orders in this period" />
              ) : (
                <div className="execution-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Part</th>
                        <th className="numeric">Orders</th>
                        <th className="numeric">Planned</th>
                        <th className="numeric">Good</th>
                        <th className="numeric">Reject</th>
                        <th className="numeric">Achievement</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.parts.map((part) => (
                        <tr key={part.partNumber}>
                          <td>
                            <strong>{part.partNumber}</strong>
                            <small>{part.partName}</small>
                          </td>
                          <td className="numeric">{part.orders}</td>
                          <td className="numeric">{quantity(part.plannedQty)}</td>
                          <td className="numeric">{quantity(part.goodQty)}</td>
                          <td className="numeric">{quantity(part.rejectQty)}</td>
                          <td className="numeric">{quantity(part.achievement)}%</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Panel>
          </div>

          {costs && (
            <Panel title="Cost per part number" note="Material, process and good quantity cover the period. Cost per piece uses cumulative finished cost (booked cost less live WIP) and cumulative good output for orders posting in the period.">
              {data.costs.length === 0 ? (
                <Empty title="No cost recorded in this period" />
              ) : (
                <div className="execution-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Part</th>
                        <th className="numeric">Orders</th>
                        <th className="numeric">Good quantity</th>
                        <th className="numeric">Material</th>
                        <th className="numeric">Process</th>
                        <th className="numeric">Total actual</th>
                        <th className="numeric">Cumulative cost per piece</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.costs.map((cost) => (
                        <tr key={`${cost.partNumber}-${cost.currency}`}>
                          <td>
                            <a href="/production-costs">{cost.partNumber}</a>
                            <small>{cost.partName}</small>
                          </td>
                          <td className="numeric">{cost.orders}</td>
                          <td className="numeric">{quantity(cost.goodQty)}</td>
                          <td className="numeric">{money(cost.materialActual, cost.currency)}</td>
                          <td className="numeric">{money(cost.processActual, cost.currency)}</td>
                          <td className="numeric"><strong>{money(cost.totalActual, cost.currency)}</strong></td>
                          <td className="numeric">{money(cost.costPerPiece, cost.currency)}{cost.finishedCost !== undefined && <small>{money(cost.finishedCost, cost.currency)} / {quantity(cost.lifetimeGoodQty)} good</small>}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Panel>
          )}
        </>
      )}
    </Workspace>
  );
}
