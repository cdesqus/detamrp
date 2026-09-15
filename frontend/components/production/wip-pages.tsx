'use client';
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';
import { request, quantity, money } from './execution-api';
import { movementLabel, type WIPMovement, type WIPOrderBalance } from './wip-api';
import {
  Badge,
  ConfirmDialog,
  Empty,
  ErrorMessage,
  Metrics,
  Panel,
  Workspace,
  date,
  dateTime,
} from './execution-ui';

const message = (error: unknown) => (error instanceof Error ? error.message : 'Unable to process this request.');
const total = (values: (string | undefined)[]) => values.reduce((sum, value) => sum + Number(value ?? 0), 0);

function MovementBadge({ movement }: { movement: WIPMovement }) {
  const label = movementLabel(movement);
  const tone = label === 'REVERSAL' ? 'cancelled' : label === 'CONSUMPTION' ? 'in_progress' : label === 'TRANSFER' ? 'released' : 'completed';
  return <span className={`execution-badge status-${tone}`}>{label}</span>;
}

export function WIPIndex() {
  const user = useCurrentUser();
  const [items, setItems] = useState<WIPOrderBalance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [view, setView] = useState('open');

  useEffect(() => {
    let alive = true;
    request<{ items: WIPOrderBalance[] }>('/production-wip')
      .then((data) => { if (alive) setItems(data.items ?? []); })
      .catch((cause) => { if (alive) setError(message(cause)); })
      .finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const costs = Boolean(user?.permissions.includes('production.report'));
  const manage = Boolean(user?.permissions.includes('production.wip'));
  const filtered = useMemo(() => items.filter((order) => {
    const matches = `${order.orderNumber} ${order.partNumber} ${order.partName} ${order.plantName}`.toLowerCase().includes(search.toLowerCase());
    if (!matches) return false;
    if (view === 'open') return Number(order.totalQty) !== 0;
    if (view === 'unreconciled') return !order.reconciled;
    return true;
  }), [items, search, view]);
  const openOrders = items.filter((order) => Number(order.totalQty) !== 0);
  const unreconciled = items.filter((order) => !order.reconciled);

  return (
    <Workspace
      title="Work in Progress"
      subtitle="Every piece between operations, with the ledger that explains how each balance came to be."
      crumbs={[{ label: 'Work in progress' }]}
    >
      <Metrics
        items={[
          { label: 'WIP quantity', value: quantity(total(items.map((order) => order.totalQty))), hint: 'Across all orders', tone: 'accent' },
          ...(costs ? [{ label: 'WIP value', value: money(total(items.map((order) => order.totalValue)), items[0]?.currency || 'IDR'), hint: 'Converted to the reporting currency' } as const] : []),
          { label: 'Orders holding WIP', value: openOrders.length, hint: `${items.length} active orders`, tone: 'amber' },
          { label: 'Unreconciled', value: unreconciled.length, hint: 'Ledger differs from entries', tone: unreconciled.length ? 'red' : 'green' },
        ]}
      />
      <ErrorMessage error={error} />
      <Panel title="WIP balances" note="Balances are the sum of the ledger: receipts less transfers, transfers less consumption.">
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
            View
            <select aria-label="View filter" value={view} onChange={(event) => setView(event.target.value)}>
              <option value="open">Orders with WIP</option>
              <option value="all">All active orders</option>
              <option value="unreconciled">Unreconciled only</option>
            </select>
          </label>
        </div>
        {loading ? (
          <p className="execution-empty" role="status">Loading WIP balances…</p>
        ) : filtered.length === 0 ? (
          <Empty
            title={error ? 'WIP balances unavailable' : 'No work in progress'}
            hint="Balances appear once an operation reports good output."
          />
        ) : (
          <div className="execution-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Production order</th>
                  <th>Part</th>
                  <th>Balance per operation</th>
                  <th className="numeric">WIP quantity</th>
                  {costs && <th className="numeric">WIP value</th>}
                  <th>Ledger</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((order) => (
                  <tr key={order.orderId}>
                    <td>
                      <a href={`/production-wip/${order.orderId}`}>{order.orderNumber}</a>
                      <small>{order.plantName}</small>
                    </td>
                    <td>
                      {order.partNumber}
                      <small>{order.partName}</small>
                    </td>
                    <td>
                      <span className="routing-flow">
                        {order.operations
                          .filter((operation) => Number(operation.balance) !== 0)
                          .map((operation) => `${operation.code} ${quantity(operation.balance)}`)
                          .join(' · ') || '—'}
                      </span>
                    </td>
                    <td className="numeric">
                      {quantity(order.totalQty)}
                      <small>{order.unitCode}</small>
                    </td>
                    {costs && <td className="numeric">{money(order.totalValue, order.currency)}</td>}
                    <td>
                      <span className={`execution-badge status-${order.reconciled ? 'completed' : 'cancelled'}`}>
                        {order.reconciled ? 'RECONCILED' : 'CHECK'}
                      </span>
                    </td>
                    <td>
                      <a className="ex-button" href={`/production-wip/${order.orderId}`}>
                        {manage ? 'Transfer WIP' : 'Open ledger'}
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="execution-pagination">
          <span>{filtered.length} of {items.length} active orders</span>
        </div>
      </Panel>
    </Workspace>
  );
}

export function WIPDetail({ id }: { id: string }) {
  const user = useCurrentUser();
  const [order, setOrder] = useState<WIPOrderBalance>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [transfer, setTransfer] = useState(false);
  const [reverse, setReverse] = useState<WIPMovement>();
  const [source, setSource] = useState('');
  const [qty, setQty] = useState('');
  const [movementDate, setMovementDate] = useState('');
  const [notes, setNotes] = useState('');
  const [reason, setReason] = useState('');

  const load = useCallback(async () => {
    try {
      setOrder(await request<WIPOrderBalance>(`/production-wip/${id}`));
    } catch (cause) {
      setError(message(cause));
    }
  }, [id]);
  useEffect(() => { void load(); }, [load]);

  const costs = Boolean(user?.permissions.includes('production.report'));
  const manage = Boolean(user?.permissions.includes('production.wip'));
  const movable = order?.operations.filter((operation) => !operation.final && Number(operation.onHand) > 0) ?? [];
  const selected = movable.find((operation) => operation.operationId === source);

  function openTransfer() {
    const first = movable[0];
    setSource(first?.operationId ?? '');
    setQty(first ? first.onHand : '');
    setMovementDate(new Date().toISOString().slice(0, 10));
    setNotes('');
    setError('');
    setTransfer(true);
  }

  async function submitTransfer(event?: FormEvent) {
    event?.preventDefault();
    if (!source) {
      setError('Select the operation holding the WIP.');
      return;
    }
    if (!Number.isFinite(Number(qty)) || Number(qty) <= 0) {
      setError('Enter a quantity greater than zero.');
      return;
    }
    if (selected && Number(qty) > Number(selected.onHand)) {
      setError('Quantity exceeds the WIP available at this operation.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const saved = await request<WIPOrderBalance>('/production-wip/transfers', {
        method: 'POST',
        body: JSON.stringify({ orderId: id, sourceOperationId: source, quantity: qty, movementDate, notes }),
      });
      setOrder(saved);
      setTransfer(false);
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }

  async function submitReversal() {
    if (!reverse) return;
    setBusy(true);
    setError('');
    try {
      const saved = await request<WIPOrderBalance>(`/production-wip/movements/${reverse.id}/reverse`, {
        method: 'POST',
        body: JSON.stringify({ reason }),
      });
      setOrder(saved);
      setReverse(undefined);
      setReason('');
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Workspace
      title={order ? `WIP · ${order.orderNumber}` : 'Work in Progress'}
      subtitle={order ? `${order.partNumber} · ${order.partName} · ${order.plantName}` : 'Loading WIP ledger…'}
      crumbs={[{ label: 'Work in progress', href: '/production-wip' }, { label: 'Order ledger' }]}
      action={
        <>
          <a className="ex-button" href="/production-wip">All balances</a>
          {order && <a className="ex-button" href={`/production-orders/${order.orderId}`}>Production order</a>}
          {order && manage && (
            <button
              type="button"
              className="ex-button primary"
              disabled={busy || movable.length === 0}
              title={movable.length === 0 ? 'No operation is holding WIP that can move yet' : undefined}
              onClick={openTransfer}
            >
              Transfer WIP
            </button>
          )}
        </>
      }
    >
      <ErrorMessage error={error} />
      {!order ? (
        !error && <p className="execution-empty" role="status">Loading WIP ledger…</p>
      ) : (
        <>
          <Metrics
            items={[
              { label: 'WIP quantity', value: quantity(order.totalQty), hint: order.unitCode, tone: 'accent' },
              ...(costs ? [{ label: 'WIP value', value: money(order.totalValue, order.currency), hint: 'At transaction cost' } as const] : []),
              {
                label: 'Waiting at operations',
                value: quantity(total(order.operations.map((operation) => operation.onHand))),
                hint: 'Not transferred yet',
                tone: 'amber',
              },
              {
                label: 'Staged for processing',
                value: quantity(total(order.operations.map((operation) => operation.staged))),
                hint: 'Transferred, not processed',
              },
            ]}
          />

          {!manage && (
            <p className="execution-footer-note">
              Moving WIP between operations needs the “Transfer work in progress” permission.
            </p>
          )}

          <div className="execution-snapshot">
            <span><small>Production order</small><a href={`/production-orders/${order.orderId}`}>{order.orderNumber}</a></span>
            <span><small>Planning</small><strong>{order.planNumber}</strong></span>
            <span><small>Order status</small><Badge status={order.status} /></span>
            <span><small>Planned quantity</small><strong>{quantity(order.plannedQty)} {order.unitCode}</strong></span>
            <span><small>Last movement</small><strong>{dateTime(order.updatedAt)}</strong></span>
          </div>

          <Panel
            title="Balance per operation"
            note="Received less transferred out is waiting; transferred in less consumed is staged for the next operation."
            action={
              <span className={`execution-badge status-${order.reconciled ? 'completed' : 'cancelled'}`}>
                {order.reconciled ? 'RECONCILED' : 'CHECK LEDGER'}
              </span>
            }
          >
            <div className="execution-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th className="numeric">Seq</th>
                    <th>Operation</th>
                    <th className="numeric">Good output</th>
                    <th className="numeric">Received</th>
                    <th className="numeric">Transferred out</th>
                    <th className="numeric">Waiting</th>
                    <th className="numeric">Transferred in</th>
                    <th className="numeric">Consumed</th>
                    <th className="numeric">Staged</th>
                    <th className="numeric">Balance</th>
                    {costs && <th className="numeric">Value</th>}
                    <th>Check</th>
                  </tr>
                </thead>
                <tbody>
                  {order.operations.map((operation) => (
                    <tr key={operation.operationId}>
                      <td className="numeric">{operation.sequence}</td>
                      <td>
                        <strong>{operation.code}</strong>
                        <small>{operation.final ? 'Final operation · output is finished production' : operation.name}</small>
                      </td>
                      <td className="numeric">{quantity(operation.goodQty)}</td>
                      <td className="numeric">{quantity(operation.received)}</td>
                      <td className="numeric">{quantity(operation.transferredOut)}</td>
                      <td className="numeric">{quantity(operation.onHand)}</td>
                      <td className="numeric">{quantity(operation.transferredIn)}</td>
                      <td className="numeric">{quantity(operation.consumed)}</td>
                      <td className="numeric">{quantity(operation.staged)}</td>
                      <td className="numeric"><strong>{quantity(operation.balance)}</strong></td>
                      {costs && <td className="numeric">{money(operation.value, order.currency)}</td>}
                      <td>
                        {operation.reconciled ? (
                          <span className="execution-badge status-completed">OK</span>
                        ) : (
                          <span className="execution-badge status-cancelled">{quantity(operation.difference)}</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Panel>

          <Panel title="Movement ledger" note="Append-only history; corrections are posted as reversals, never as edits.">
            {order.movements.length === 0 ? (
              <Empty title="No movements recorded" hint="Good output of a non-final operation creates the first receipt." />
            ) : (
              <div className="execution-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Movement</th>
                      <th>From → to</th>
                      <th className="numeric">Quantity</th>
                      {costs && <th className="numeric">Unit cost</th>}
                      {costs && <th className="numeric">Total</th>}
                      <th>Reference</th>
                      <th>Recorded by</th>
                      {manage && <th>Actions</th>}
                    </tr>
                  </thead>
                  <tbody>
                    {order.movements.map((movement) => (
                      <tr key={movement.id}>
                        <td>{date(movement.movementDate)}</td>
                        <td><MovementBadge movement={movement} /></td>
                        <td>
                          <span className="routing-flow">
                            {movement.sourceCode || 'Production'} → {movement.destinationCode || 'Processed'}
                          </span>
                          {movement.notes && <small>{movement.notes}</small>}
                        </td>
                        <td className="numeric">{quantity(movement.quantity)}</td>
                        {costs && <td className="numeric">{money(movement.unitCost, movement.currency)}</td>}
                        {costs && <td className="numeric">{money(movement.totalCost, movement.currency)}</td>}
                        <td>
                          {movement.entryNumber || '—'}
                          {movement.reversed && <small>reversed</small>}
                        </td>
                        <td>
                          {movement.createdBy}
                          <small>{dateTime(movement.createdAt)}</small>
                        </td>
                        {manage && (
                          <td>
                            {movement.canReverse ? (
                              <button
                                type="button"
                                className="ex-button danger"
                                aria-label={`Reverse transfer of ${quantity(movement.quantity)}`}
                                onClick={() => { setReverse(movement); setReason(''); }}
                              >
                                Reverse
                              </button>
                            ) : (
                              <span className="muted">—</span>
                            )}
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Panel>

          {transfer && (
            <ConfirmDialog
              title="Transfer WIP to the next operation"
              confirmLabel="Post transfer"
              cancelLabel="Cancel"
              busy={busy}
              onConfirm={() => void submitTransfer()}
              onClose={() => setTransfer(false)}
            >
              <p>Move finished pieces to the next operation. Only transferred stock can be processed there.</p>
              <label>
                From operation
                <select aria-label="Source operation" value={source} onChange={(event) => {
                  setSource(event.target.value);
                  const next = movable.find((operation) => operation.operationId === event.target.value);
                  setQty(next ? next.onHand : '');
                }}>
                  {movable.map((operation) => (
                    <option key={operation.operationId} value={operation.operationId}>
                      {operation.code} · {quantity(operation.onHand)} {order.unitCode} waiting
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Quantity
                <input
                  aria-label="Transfer quantity"
                  type="number"
                  min="0.000001"
                  step="any"
                  value={qty}
                  onChange={(event) => setQty(event.target.value)}
                />
                <small>Up to {quantity(selected?.onHand ?? '0')} {order.unitCode} is waiting at this operation.</small>
              </label>
              <label>
                Movement date
                <input aria-label="Movement date" type="date" value={movementDate} onChange={(event) => setMovementDate(event.target.value)} />
              </label>
              <label>
                Notes
                <textarea rows={2} value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Trolley, batch or handover note" />
              </label>
            </ConfirmDialog>
          )}

          {reverse && (
            <ConfirmDialog
              title="Reverse WIP transfer"
              confirmLabel="Post reversal"
              cancelLabel="Keep movement"
              tone="danger"
              busy={busy}
              onConfirm={() => void submitReversal()}
              onClose={() => setReverse(undefined)}
            >
              <p>
                Return {quantity(reverse.quantity)} {order.unitCode} from {reverse.destinationCode} to {reverse.sourceCode}.
                The original movement stays in the ledger with its reversal beside it.
              </p>
              <label>
                Reason
                <textarea rows={2} value={reason} onChange={(event) => setReason(event.target.value)} />
              </label>
            </ConfirmDialog>
          )}
        </>
      )}
    </Workspace>
  );
}
