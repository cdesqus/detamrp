'use client';
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { useCurrentUser } from '../app-shell/app-shell';
import { request, quantity, money, type Order, type OrderOptions } from './execution-api';
import {
  Badge,
  ConfirmDialog,
  Empty,
  ErrorMessage,
  Metrics,
  Panel,
  Progress,
  Tabs,
  Workspace,
  date,
  dateTime,
  percent,
} from './execution-ui';

const PAGE_SIZE = 15;
const ORDER_STATUSES = ['RELEASED', 'IN_PROGRESS', 'PARTIAL', 'COMPLETED', 'CANCELLED'];
const ACTIVE_STATUSES = ['RELEASED', 'IN_PROGRESS', 'PARTIAL'];

const message = (error: unknown) => (error instanceof Error ? error.message : 'Unable to process this request.');

/** An order may only be changed while it carries no execution history. */
const editable = (order: Order) =>
  ['RELEASED', 'IN_PROGRESS'].includes(order.status) && order.entryCount === 0 && Number(order.wipQty) === 0;

const overdue = (order: Order) =>
  ACTIVE_STATUSES.includes(order.status) && new Date(`${order.dueDate.slice(0, 10)}T23:59:59`) < new Date();

export function OrderIndex() {
  const user = useCurrentUser();
  const [items, setItems] = useState<Order[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);

  useEffect(() => {
    let alive = true;
    request<{ items: Order[] }>('/production-orders')
      .then((data) => { if (alive) setItems(data.items ?? []); })
      .catch((cause) => { if (alive) setError(message(cause)); })
      .finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const filtered = useMemo(() => items.filter((order) => {
    const haystack = `${order.orderNumber} ${order.planNumber} ${order.partNumber} ${order.partName} ${order.plantName}`;
    return (!status || order.status === status) && haystack.toLowerCase().includes(search.toLowerCase());
  }), [items, status, search]);

  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const current = Math.min(page, pages);
  const visible = filtered.slice((current - 1) * PAGE_SIZE, current * PAGE_SIZE);
  const count = (value: string) => items.filter((order) => order.status === value).length;

  return (
    <Workspace
      title="Production Orders"
      subtitle="Release approved plans to the floor, then follow output, rejects and work in progress per order."
      crumbs={[{ label: 'Production orders' }]}
      action={user?.permissions.includes('production.order') && (
        <a className="ex-button primary" href="/production-orders/new">New Production Order</a>
      )}
    >
      <Metrics
        items={[
          { label: 'Total orders', value: items.length, hint: 'All production orders' },
          { label: 'Awaiting release to floor', value: count('RELEASED'), hint: 'Ready to start', tone: 'accent' },
          { label: 'In progress', value: count('IN_PROGRESS') + count('PARTIAL'), hint: 'Running or paused', tone: 'amber' },
          { label: 'Completed', value: count('COMPLETED'), hint: 'Production finished', tone: 'green' },
        ]}
      />
      <ErrorMessage error={error} />
      <Panel title="Order register" note="Every order carries the BOM, routing and cost snapshot taken at release.">
        <div className="execution-filters">
          <label>
            Search orders
            <input
              type="search"
              placeholder="Order, planning, part or plant"
              value={search}
              onChange={(event) => { setSearch(event.target.value); setPage(1); }}
            />
          </label>
          <label>
            Status
            <select
              aria-label="Status filter"
              value={status}
              onChange={(event) => { setStatus(event.target.value); setPage(1); }}
            >
              <option value="">All statuses</option>
              {ORDER_STATUSES.map((value) => <option key={value} value={value}>{value.replaceAll('_', ' ')}</option>)}
            </select>
          </label>
        </div>
        {loading ? (
          <p className="execution-empty" role="status">Loading production orders…</p>
        ) : visible.length === 0 ? (
          <Empty
            title={error ? 'Production orders unavailable' : 'No production orders match this view'}
            hint="Adjust the filters or release an approved planning line."
          />
        ) : (
          <div className="execution-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Production order</th>
                  <th>Output part</th>
                  <th>Plant / due date</th>
                  <th className="numeric">Planned</th>
                  <th className="numeric">Good / reject</th>
                  <th className="numeric">WIP</th>
                  <th>Progress</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {visible.map((order) => (
                  <tr key={order.id}>
                    <td>
                      <a href={`/production-orders/${order.id}`}>{order.orderNumber}</a>
                      <small>{order.planNumber}</small>
                    </td>
                    <td>
                      {order.partNumber}
                      <small>{order.partName}</small>
                    </td>
                    <td>
                      {order.plantName}
                      <small>{date(order.dueDate)}{overdue(order) ? ' · overdue' : ''}</small>
                    </td>
                    <td className="numeric">
                      {quantity(order.plannedQty)}
                      <small>{order.unitCode}</small>
                    </td>
                    <td className="numeric">{quantity(order.actualGood)} / {quantity(order.rejectQty)}</td>
                    <td className="numeric">{quantity(order.wipQty)}</td>
                    <td><Progress value={percent(order.actualGood, order.plannedQty)} /></td>
                    <td><Badge status={order.status} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="execution-pagination">
          <span>{filtered.length} of {items.length} orders · page {current} of {pages}</span>
          <div className="execution-actions">
            <button type="button" className="ex-button" disabled={current === 1} onClick={() => setPage(current - 1)}>Previous</button>
            <button type="button" className="ex-button" disabled={current === pages} onClick={() => setPage(current + 1)}>Next</button>
          </div>
        </div>
      </Panel>
    </Workspace>
  );
}

export function OrderForm({ id, planId }: { id?: string; planId?: string }) {
  const user = useCurrentUser();
  const router = useRouter();
  const [options, setOptions] = useState<OrderOptions>();
  const [order, setOrder] = useState<Order>();
  const [lineId, setLineId] = useState('');
  const [qty, setQty] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [notes, setNotes] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        if (id) {
          const data = await request<Order>(`/production-orders/${id}`);
          if (!alive) return;
          setOrder(data);
          setQty(data.plannedQty);
          setDueDate(data.dueDate.slice(0, 10));
          setNotes(data.notes || '');
        } else {
          const data = await request<OrderOptions>('/production-orders/options');
          if (!alive) return;
          setOptions(data);
          const first = data.lines.find((line) => line.planId === planId && Number(line.remainingQty) > 0);
          if (first) {
            setLineId(first.id);
            setDueDate(first.periodEnd.slice(0, 10));
          }
        }
      } catch (cause) {
        if (alive) setError(message(cause));
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => { alive = false; };
  }, [id, planId]);

  const lines = options?.lines.filter((line) => Number(line.remainingQty) > 0) ?? [];
  const line = lines.find((candidate) => candidate.id === lineId);
  const source = line ?? order;
  const canEdit = !order || editable(order);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError('');
    if (!qty.trim() || !Number.isFinite(Number(qty)) || Number(qty) <= 0) {
      setError('Enter a planned quantity greater than zero.');
      return;
    }
    if (!id && !line) {
      setError('Select an approved planning line.');
      return;
    }
    if (line && Number(qty) > Number(line.remainingQty)) {
      setError('Planned quantity exceeds the available planning balance.');
      return;
    }
    if (!dueDate) {
      setError('Choose a due date.');
      return;
    }
    setBusy(true);
    try {
      const saved = await request<Order>(id ? `/production-orders/${id}` : '/production-orders', {
        method: id ? 'PUT' : 'POST',
        body: JSON.stringify({ ...(id ? {} : { planLineId: lineId }), plannedQty: qty, dueDate, notes }),
      });
      router.push(`/production-orders/${saved.id}`);
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }

  const backHref = id ? `/production-orders/${id}` : '/production-orders';

  return (
    <Workspace
      title={id ? 'Edit Production Order' : 'New Production Order'}
      subtitle="Turn an approved planning balance into a production order with its own BOM and cost snapshot."
      crumbs={[{ label: 'Production orders', href: '/production-orders' }, { label: id ? 'Edit' : 'New order' }]}
      action={<a className="ex-button" href={backHref}>Back to orders</a>}
    >
      <ErrorMessage error={error} />
      {loading ? (
        <p className="execution-empty" role="status">Loading order setup…</p>
      ) : !user?.permissions.includes('production.order') ? (
        <Panel title="Access restricted">
          <div className="execution-panel-body">
            <p>Production order permission is required to release or change orders.</p>
          </div>
        </Panel>
      ) : !canEdit ? (
        <Panel title="Order is locked">
          <div className="execution-panel-body">
            <p>Only active orders without daily entries or work in progress can be edited.</p>
          </div>
        </Panel>
      ) : (
        <form onSubmit={submit}>
          <Panel
            title={id ? 'Order details' : 'Planning allocation'}
            note={id ? 'Scheduling changes keep the original BOM and cost snapshot.' : 'Only approved planning lines with an open balance can be released.'}
          >
            <div className="execution-panel-body execution-form-grid">
              {!id && (
                <label className="full">
                  Approved planning line
                  <select
                    required
                    value={lineId}
                    onChange={(event) => {
                      setLineId(event.target.value);
                      const selected = lines.find((candidate) => candidate.id === event.target.value);
                      setDueDate(selected?.periodEnd.slice(0, 10) || '');
                    }}
                  >
                    <option value="">Select planning and output part</option>
                    {lines.map((option) => (
                      <option value={option.id} key={option.id}>
                        {option.planNumber} · {option.partNumber} · {option.plantName}
                      </option>
                    ))}
                  </select>
                  {!lines.length && <small>No approved planning lines have an available balance. Approve a production plan first.</small>}
                </label>
              )}
              {source && (
                <div className="execution-snapshot full">
                  <span><small>Output</small><strong>{source.partNumber}</strong>{source.partName}</span>
                  <span><small>Plant</small><strong>{source.plantName}</strong></span>
                  <span><small>Planning</small><strong>{source.planNumber}</strong></span>
                  {line && <span><small>Allocation balance</small><strong>Available: {quantity(line.remainingQty)} {line.unitCode}</strong></span>}
                  {order && <span><small>BOM snapshot</small><strong>Revision {order.bomRevision}</strong></span>}
                </div>
              )}
              <label>
                Planned quantity
                <input
                  aria-label="Planned quantity"
                  type="number"
                  min="0.000001"
                  step="any"
                  required
                  value={qty}
                  onChange={(event) => setQty(event.target.value)}
                />
                <small>Quantity in {source?.unitCode || 'the output unit'}.</small>
              </label>
              <label>
                Due date
                <input type="date" required value={dueDate} onChange={(event) => setDueDate(event.target.value)} />
                <small>Defaults to the end of the planning period.</small>
              </label>
              <label className="full">
                Notes
                <textarea
                  rows={4}
                  value={notes}
                  onChange={(event) => setNotes(event.target.value)}
                  placeholder="Production instructions or scheduling notes"
                />
              </label>
            </div>
            <footer className="execution-form-footer">
              <span className="execution-footer-note">
                Material requirements, routing operations and rates are captured from the active master data when the order is released.
              </span>
              <a className="ex-button" href={backHref}>Cancel</a>
              <button type="submit" className="ex-button primary" disabled={busy || (!id && !line) || Boolean(id && !order)}>
                {busy ? 'Saving…' : id ? 'Save changes' : 'Release Production Order'}
              </button>
            </footer>
          </Panel>
        </form>
      )}
    </Workspace>
  );
}

export function OrderDetail({ id }: { id: string }) {
  const user = useCurrentUser();
  const router = useRouter();
  const [order, setOrder] = useState<Order>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState('Operations');
  const [dialog, setDialog] = useState<'' | 'cancel' | 'delete'>('');
  const [reason, setReason] = useState('');

  const load = useCallback(async () => {
    try {
      setOrder(await request<Order>(`/production-orders/${id}`));
    } catch (cause) {
      setError(message(cause));
    }
  }, [id]);
  useEffect(() => { void load(); }, [load]);

  async function act(kind: 'start' | 'cancel' | 'delete') {
    if (kind === 'cancel' && !reason.trim()) {
      setError('A cancellation reason is required.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await request(`/production-orders/${id}${kind === 'delete' ? '' : `/${kind}`}`, {
        method: kind === 'delete' ? 'DELETE' : 'POST',
        ...(kind === 'delete' ? {} : { body: JSON.stringify(kind === 'cancel' ? { reason } : {}) }),
      });
      if (kind === 'delete') {
        router.push('/production-orders');
        return;
      }
      setDialog('');
      await load();
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }

  const manage = user?.permissions.includes('production.order');
  const costs = user?.permissions.includes('production.report');
  const active = order && ACTIVE_STATUSES.includes(order.status);
  const totalEstimate = order ? Number(order.materialEstimate) + Number(order.processEstimate) : 0;
  const totalActual = order ? Number(order.materialCost) + Number(order.processCost) : 0;

  return (
    <Workspace
      title={order?.orderNumber || 'Production Order'}
      subtitle={order ? `${order.partNumber} · ${order.partName} · ${order.plantName}` : 'Loading production order details…'}
      crumbs={[{ label: 'Production orders', href: '/production-orders' }, { label: 'Order detail' }]}
      action={
        <>
          <a className="ex-button" href="/production-orders">All orders</a>
          {order && <Badge status={order.status} />}
        </>
      }
    >
      <ErrorMessage error={error} />
      {!order ? (
        !error && <p className="execution-empty" role="status">Loading production order…</p>
      ) : (
        <>
          <div className="execution-actions">
            {manage && editable(order) && (
              <>
                <a className="ex-button" href={`/production-orders/${id}/edit`}>Edit order</a>
                <button type="button" className="ex-button danger" disabled={busy} onClick={() => setDialog('delete')}>Delete order</button>
              </>
            )}
            {manage && ['RELEASED', 'PARTIAL'].includes(order.status) && (
              <button type="button" className="ex-button primary" disabled={busy} onClick={() => void act('start')}>
                {order.status === 'PARTIAL' ? 'Resume production' : 'Start production'}
              </button>
            )}
            {user?.permissions.includes('production.entry') && ['RELEASED', 'IN_PROGRESS'].includes(order.status) && (
              <a className="ex-button primary" href={`/daily-production/new?orderId=${id}`}>New Daily Entry</a>
            )}
            {manage && active && Number(order.wipQty) === 0 && (
              <button type="button" className="ex-button danger" disabled={busy} onClick={() => setDialog('cancel')}>Cancel order</button>
            )}
            <a className="ex-button" href={`/production-wip/${id}`}>WIP ledger</a>
            {user?.permissions.includes('production.report') && (
              <>
                <a className="ex-button" href={`/production-costs/${id}`}>Cost breakdown</a>
                <a className="ex-button" href={`/api/production-reports/orders/${id}/execution.xlsx`}>Export Excel</a>
                <a className="ex-button" href={`/api/production-reports/orders/${id}/execution.pdf`}>Export PDF</a>
              </>
            )}
          </div>

          <Metrics
            items={[
              { label: 'Planned quantity', value: quantity(order.plannedQty), hint: order.unitCode },
              {
                label: 'Actual good',
                value: quantity(order.actualGood),
                hint: `${percent(order.actualGood, order.plannedQty).toFixed(1)}% of target`,
                tone: 'green',
              },
              {
                label: 'Reject quantity',
                value: quantity(order.rejectQty),
                hint: `${percent(order.rejectQty, order.actualGood).toFixed(1)}% reject rate`,
                tone: 'red',
              },
              {
                label: 'Work in progress',
                value: quantity(order.wipQty),
                hint: `${order.entryCount} daily entries`,
                tone: 'amber',
              },
            ]}
          />

          <div className="execution-snapshot">
            <span>
              <small>Production planning</small>
              <a href={`/production-planning/${order.planId}`}>{order.planNumber}</a>
            </span>
            <span><small>Period start</small><strong>{date(order.periodStart)}</strong></span>
            <span>
              <small>Due date</small>
              <strong>{date(order.dueDate)}{overdue(order) ? ' · overdue' : ''}</strong>
            </span>
            <span><small>BOM revision</small><strong>{order.bomRevision}</strong></span>
            <span><small>Created by</small><strong>{order.createdBy}</strong></span>
            <span><small>Last updated</small><strong>{dateTime(order.updatedAt)}</strong></span>
          </div>

          <Panel title="Execution detail" note="Routing progress, frozen snapshots and cost performance for this order.">
            <Tabs
              tabs={['Operations', 'Material snapshot', ...(costs ? ['Costs'] : []), 'History']}
              active={tab}
              onSelect={setTab}
              label="Order details"
            />
            <div role="tabpanel">
              {tab === 'Operations' && (order.operations.length ? (
                <div className="execution-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Seq</th>
                        <th>Operation</th>
                        <th className="numeric">Planned</th>
                        <th className="numeric">Processed</th>
                        <th className="numeric">Good</th>
                        <th className="numeric">Reject</th>
                        <th className="numeric">WIP</th>
                        {costs && <th className="numeric">Rate</th>}
                        <th>Progress</th>
                      </tr>
                    </thead>
                    <tbody>
                      {order.operations.map((operation) => (
                        <tr key={operation.id}>
                          <td>{operation.sequence}</td>
                          <td>
                            <strong>{operation.code}</strong>
                            <small>{operation.name}</small>
                          </td>
                          <td className="numeric">{quantity(operation.plannedQty)}</td>
                          <td className="numeric">{quantity(operation.processedQty)}</td>
                          <td className="numeric">{quantity(operation.goodQty)}</td>
                          <td className="numeric">{quantity(operation.rejectQty)}</td>
                          <td className="numeric">{quantity(operation.wipQty)}</td>
                          {costs && <td className="numeric">{money(operation.rate, order.currency)}</td>}
                          <td><Progress value={percent(operation.goodQty, operation.plannedQty)} /></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <Empty title="No operations in this order snapshot" hint="Define a routing for this part before releasing production." />
              ))}

              {tab === 'Material snapshot' && (order.materials.length ? (
                <div className="execution-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Material</th>
                        <th className="numeric">Usage per output unit</th>
                        <th>Unit</th>
                        {costs && <th className="numeric">Snapshot price</th>}
                        <th className="numeric">Order requirement</th>
                      </tr>
                    </thead>
                    <tbody>
                      {order.materials.map((material) => (
                        <tr key={material.id}>
                          <td>
                            <strong>{material.partNumber}</strong>
                            <small>{material.partName}</small>
                          </td>
                          <td className="numeric">{quantity(material.usageQty)}</td>
                          <td>{material.unitCode}</td>
                          {costs && <td className="numeric">{money(material.unitPrice, material.currency)}</td>}
                          <td className="numeric">
                            {quantity(Number(material.usageQty) * Number(order.plannedQty))}
                            <small>{material.unitCode}</small>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <Empty title="No materials in this order snapshot" hint="Activate a BOM revision for this part to capture material requirements." />
              ))}

              {costs && tab === 'Costs' && (
                <div className="execution-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Cost component</th>
                        <th className="numeric">Estimate at release</th>
                        <th className="numeric">Actual</th>
                        <th className="numeric">Variance</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr>
                        <td>Materials</td>
                        <td className="numeric">{money(order.materialEstimate, order.currency)}</td>
                        <td className="numeric">{money(order.materialCost, order.currency)}</td>
                        <td className="numeric">{money(Number(order.materialCost) - Number(order.materialEstimate), order.currency)}</td>
                      </tr>
                      <tr>
                        <td>Processes</td>
                        <td className="numeric">{money(order.processEstimate, order.currency)}</td>
                        <td className="numeric">{money(order.processCost, order.currency)}</td>
                        <td className="numeric">{money(Number(order.processCost) - Number(order.processEstimate), order.currency)}</td>
                      </tr>
                    </tbody>
                    <tfoot>
                      <tr>
                        <td>Total cost</td>
                        <td className="numeric">{money(totalEstimate, order.currency)}</td>
                        <td className="numeric">{money(totalActual, order.currency)}</td>
                        <td className="numeric">{money(totalActual - totalEstimate, order.currency)}</td>
                      </tr>
                    </tfoot>
                  </table>
                </div>
              )}

              {tab === 'History' && (order.history.length ? (
                <ol className="execution-history">
                  {order.history.map((entry, index) => (
                    <li key={index}>
                      <strong>{entry.action.replaceAll('_', ' ')}</strong>
                      <small>{entry.actor} · {dateTime(entry.occurredAt)}</small>
                    </li>
                  ))}
                </ol>
              ) : (
                <Empty title="No history recorded" hint="Order events appear here as production progresses." />
              ))}
            </div>
          </Panel>

          {order.notes && (
            <Panel title="Production notes">
              <div className="execution-panel-body">
                <p style={{ whiteSpace: 'pre-wrap' }}>{order.notes}</p>
              </div>
            </Panel>
          )}

          {dialog === 'cancel' && (
            <ConfirmDialog
              title="Cancel production order"
              confirmLabel="Confirm cancellation"
              cancelLabel="Keep order"
              tone="danger"
              busy={busy}
              onConfirm={() => void act('cancel')}
              onClose={() => setDialog('')}
            >
              <p>Cancelling {order.orderNumber} stops further production entries and releases its planning allocation.</p>
              <label>
                Cancellation reason
                <textarea required rows={3} value={reason} onChange={(event) => setReason(event.target.value)} />
              </label>
            </ConfirmDialog>
          )}
          {dialog === 'delete' && (
            <ConfirmDialog
              title="Delete production order"
              confirmLabel="Confirm deletion"
              cancelLabel="Keep order"
              tone="danger"
              busy={busy}
              onConfirm={() => void act('delete')}
              onClose={() => setDialog('')}
            >
              <p>Delete {order.orderNumber} and return its allocation to planning? This is only possible while the order has no entries.</p>
            </ConfirmDialog>
          )}
        </>
      )}
    </Workspace>
  );
}
