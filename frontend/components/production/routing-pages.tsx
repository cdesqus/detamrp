'use client';
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { useCurrentUser } from '../app-shell/app-shell';
import { request, money, type Routing, type RoutingOptions, type RoutingStep } from './execution-api';
import {
  ConfirmDialog,
  Empty,
  ErrorMessage,
  Metrics,
  Panel,
  Workspace,
  dateTime,
} from './execution-ui';

const message = (error: unknown) => (error instanceof Error ? error.message : 'Unable to process this request.');
const emptyStep = (): RoutingStep => ({ code: '', name: '', rate: '' });
const stepTotal = (steps: RoutingStep[]) => steps.reduce((sum, step) => sum + (Number(step.rate) || 0), 0);

function RoutingState({ active }: { active: boolean }) {
  return <span className={`execution-badge status-${active ? 'completed' : 'cancelled'}`}>{active ? 'ACTIVE' : 'INACTIVE'}</span>;
}

/** Operation codes in sequence, rendered as the flow the shop floor follows. */
function Flow({ steps }: { steps: RoutingStep[] }) {
  return <span className="routing-flow">{steps.map((step) => step.code).join(' → ') || '—'}</span>;
}

export function RoutingIndex() {
  const user = useCurrentUser();
  const costs = Boolean(user?.permissions.includes('production.report'));
  const canWrite = Boolean(user?.permissions.includes('production.routing') && costs);
  const [items, setItems] = useState<Routing[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [state, setState] = useState('');

  useEffect(() => {
    let alive = true;
    request<{ items: Routing[] }>('/production-routings')
      .then((data) => { if (alive) setItems(data.items ?? []); })
      .catch((cause) => { if (alive) setError(message(cause)); })
      .finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const filtered = useMemo(() => items.filter((routing) => {
    const haystack = `${routing.partNumber} ${routing.partName} ${routing.name} ${routing.steps.map((step) => step.code).join(' ')}`;
    return (!state || String(routing.active) === state) && haystack.toLowerCase().includes(search.toLowerCase());
  }), [items, state, search]);
  const activeCount = items.filter((routing) => routing.active).length;
  const parts = new Set(items.filter((routing) => routing.active).map((routing) => routing.partId)).size;

  return (
    <Workspace
      title="Routing & Operations"
      subtitle="Define the operation sequence and cost per piece that every production order snapshots at release."
      crumbs={[{ label: 'Routing' }]}
      action={canWrite && (
        <a className="ex-button primary" href="/production-routings/new">New Routing</a>
      )}
    >
      <Metrics
        items={[
          { label: 'Routing revisions', value: items.length, hint: 'All parts' },
          { label: 'Active routings', value: activeCount, hint: 'Usable by new orders', tone: 'accent' },
          { label: 'Parts covered', value: parts, hint: 'Parts that can be produced', tone: 'green' },
          { label: 'Draft revisions', value: items.length - activeCount, hint: 'Awaiting activation', tone: 'amber' },
        ]}
      />
      <ErrorMessage error={error} />
      <Panel title="Routing register" note="One routing per part is active; earlier revisions stay readable for traceability.">
        <div className="execution-filters">
          <label>
            Search routings
            <input
              type="search"
              placeholder="Part, routing name or operation code"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </label>
          <label>
            State
            <select aria-label="State filter" value={state} onChange={(event) => setState(event.target.value)}>
              <option value="">All routings</option>
              <option value="true">Active</option>
              <option value="false">Inactive</option>
            </select>
          </label>
        </div>
        {loading ? (
          <p className="execution-empty" role="status">Loading routings…</p>
        ) : filtered.length === 0 ? (
          <Empty
            title={error ? 'Routings unavailable' : 'No routings match this view'}
            hint="Define a routing before releasing production orders for a part."
          />
        ) : (
          <div className="execution-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Part</th>
                  <th>Routing</th>
                  <th className="numeric">Rev</th>
                  <th>Operation flow</th>
                  {costs && <th className="numeric">Cost per piece</th>}
                  <th className="numeric">Orders</th>
                  <th>State</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((routing) => (
                  <tr key={routing.id}>
                    <td>
                      <a href={`/production-routings/${routing.id}`}>{routing.partNumber}</a>
                      <small>{routing.partName}</small>
                    </td>
                    <td>
                      {routing.name}
                      <small>{routing.kind === 'FG' ? 'Finished good' : 'Process material'}</small>
                    </td>
                    <td className="numeric">{routing.revision}</td>
                    <td><Flow steps={routing.steps} /></td>
                    {costs && <td className="numeric">{money(routing.totalRate, routing.currency)}</td>}
                    <td className="numeric">{routing.linkedOrders}</td>
                    <td><RoutingState active={routing.active} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="execution-pagination">
          <span>{filtered.length} of {items.length} routing revisions</span>
        </div>
      </Panel>
    </Workspace>
  );
}

export function RoutingForm({ id, copyFrom }: { id?: string; copyFrom?: string }) {
  const user = useCurrentUser();
  const costs = Boolean(user?.permissions.includes('production.report'));
  const canWrite = Boolean(user?.permissions.includes('production.routing') && costs);
  const router = useRouter();
  const [options, setOptions] = useState<RoutingOptions>();
  const [routing, setRouting] = useState<Routing>();
  const [partId, setPartId] = useState('');
  const [kind, setKind] = useState('FG');
  const [name, setName] = useState('');
  const [currency, setCurrency] = useState('IDR');
  const [steps, setSteps] = useState<RoutingStep[]>([emptyStep()]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const [loadedOptions, existing] = await Promise.all([
          request<RoutingOptions>('/production-routings/options'),
          id || copyFrom ? request<Routing>(`/production-routings/${id ?? copyFrom}`) : Promise.resolve(undefined),
        ]);
        if (!alive) return;
        setOptions(loadedOptions);
        if (existing) {
          setRouting(id ? existing : undefined);
          setPartId(existing.partId);
          setKind(existing.kind);
          setName(id ? existing.name : `${existing.name} (revision ${existing.revision + 1})`);
          setCurrency(existing.currency);
          setSteps(existing.steps.map((step) => ({ ...step })));
        }
      } catch (cause) {
        if (alive) setError(message(cause));
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => { alive = false; };
  }, [id, copyFrom]);

  function change(index: number, patch: Partial<RoutingStep>) {
    setSteps((current) => current.map((step, position) => (position === index ? { ...step, ...patch } : step)));
  }

  function move(index: number, offset: number) {
    setSteps((current) => {
      const target = index + offset;
      if (target < 0 || target >= current.length) return current;
      const next = [...current];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError('');
    if (!id && !partId) {
      setError('Select the part this routing produces.');
      return;
    }
    if (!name.trim()) {
      setError('Enter a routing name.');
      return;
    }
    const cleaned = steps.map((step) => ({ ...step, code: step.code.trim().toUpperCase(), name: step.name.trim() }));
    if (cleaned.some((step) => !step.code || !step.name)) {
      setError('Every operation needs a code and a name.');
      return;
    }
    if (new Set(cleaned.map((step) => step.code)).size !== cleaned.length) {
      setError('Operation codes must be unique within a routing.');
      return;
    }
    if (cleaned.some((step) => !Number.isFinite(Number(step.rate)) || Number(step.rate) < 0)) {
      setError('Enter a cost per piece of zero or more for every operation.');
      return;
    }
    setBusy(true);
    try {
      const saved = await request<Routing>(id ? `/production-routings/${id}` : '/production-routings', {
        method: id ? 'PUT' : 'POST',
        body: JSON.stringify({
          ...(id ? {} : { partId, kind }),
          name: name.trim(),
          currency,
          steps: cleaned.map((step) => ({ ...step, rate: step.rate || '0' })),
        }),
      });
      router.push(`/production-routings/${saved.id}`);
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }

  const backHref = id ? `/production-routings/${id}` : '/production-routings';
  const parts = options?.parts.filter((part) => part.kind === kind) ?? [];
  const locked = Boolean(routing && routing.linkedOrders > 0);

  return (
    <Workspace
      title={id ? 'Edit Routing' : 'New Routing'}
      subtitle="Operations run in sequence: the good output of one operation becomes the input of the next."
      crumbs={[{ label: 'Routing', href: '/production-routings' }, { label: id ? 'Edit' : 'New routing' }]}
      action={<a className="ex-button" href={backHref}>Back to routings</a>}
    >
      <ErrorMessage error={error} />
      {loading ? (
        <p className="execution-empty" role="status">Loading routing setup…</p>
      ) : !canWrite ? (
        <Panel title="Access restricted">
          <div className="execution-panel-body"><p>Routing and production report permissions are required to define operations.</p></div>
        </Panel>
      ) : locked ? (
        <Panel title="Routing is frozen">
          <div className="execution-panel-body">
            <p>Production orders already reference this routing. Create a new revision instead of changing a frozen one.</p>
          </div>
        </Panel>
      ) : (
        <form onSubmit={submit}>
          <Panel title="Routing header" note={id ? 'The output part of an existing routing cannot be changed.' : 'Each part keeps one active routing; new revisions start inactive.'}>
            <div className="execution-panel-body execution-form-grid">
              <label>
                Output type
                <select
                  aria-label="Output type"
                  disabled={Boolean(id)}
                  value={kind}
                  onChange={(event) => { setKind(event.target.value); setPartId(''); }}
                >
                  <option value="FG">Finished Good</option>
                  <option value="RAW_MATERIAL">Process Material</option>
                </select>
              </label>
              <label>
                Part
                <select
                  aria-label="Part"
                  required={!id}
                  disabled={Boolean(id)}
                  value={partId}
                  onChange={(event) => setPartId(event.target.value)}
                >
                  <option value="">Select active part</option>
                  {parts.map((part) => (
                    <option key={part.id} value={part.id}>{part.partNumber} — {part.partName}</option>
                  ))}
                </select>
              </label>
              <label>
                Routing name
                <input aria-label="Routing name" required value={name} onChange={(event) => setName(event.target.value)} />
              </label>
              <label>
                Currency
                <input
                  aria-label="Currency"
                  required
                  maxLength={3}
                  value={currency}
                  onChange={(event) => setCurrency(event.target.value.toUpperCase())}
                />
                <small>Operation rates and BOM prices must share this currency.</small>
              </label>
            </div>
          </Panel>

          <Panel
            title="Operation sequence"
            note="Sequence numbers follow the row order. Welding cannot start before stamping output exists."
            action={
              <button type="button" className="ex-button" onClick={() => setSteps((current) => [...current, emptyStep()])}>
                + Add operation
              </button>
            }
          >
            <div className="execution-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th className="numeric">Seq</th>
                    <th>Operation code</th>
                    <th>Operation name</th>
                    {costs && <th className="numeric">Cost per piece</th>}
                    <th>Order</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {steps.map((step, index) => (
                    <tr key={index}>
                      <td className="numeric">{index + 1}</td>
                      <td>
                        <input
                          aria-label={`Operation code ${index + 1}`}
                          required
                          maxLength={20}
                          placeholder="STAMPING"
                          value={step.code}
                          onChange={(event) => change(index, { code: event.target.value.toUpperCase() })}
                        />
                      </td>
                      <td>
                        <input
                          aria-label={`Operation name ${index + 1}`}
                          required
                          value={step.name}
                          onChange={(event) => change(index, { name: event.target.value })}
                        />
                      </td>
                      <td className="numeric">
                        <input
                          aria-label={`Cost per piece ${index + 1}`}
                          type="number"
                          min="0"
                          step="any"
                          value={step.rate}
                          onChange={(event) => change(index, { rate: event.target.value })}
                        />
                      </td>
                      <td>
                        <div className="execution-actions">
                          <button
                            type="button"
                            className="ex-button"
                            aria-label={`Move operation ${index + 1} up`}
                            disabled={index === 0}
                            onClick={() => move(index, -1)}
                          >
                            ↑
                          </button>
                          <button
                            type="button"
                            className="ex-button"
                            aria-label={`Move operation ${index + 1} down`}
                            disabled={index === steps.length - 1}
                            onClick={() => move(index, 1)}
                          >
                            ↓
                          </button>
                        </div>
                      </td>
                      <td>
                        <button
                          type="button"
                          className="ex-button danger"
                          aria-label={`Remove operation ${index + 1}`}
                          disabled={steps.length === 1}
                          onClick={() => setSteps((current) => current.filter((_, position) => position !== index))}
                        >
                          Remove
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={3}>Process cost per piece · {steps.length} operations</td>
                    <td className="numeric">{money(stepTotal(steps), currency || 'IDR')}</td>
                    <td colSpan={2} />
                  </tr>
                </tfoot>
              </table>
            </div>
            <footer className="execution-form-footer">
              <span className="execution-footer-note">Released orders keep the rates captured at release, so later changes never rewrite history.</span>
              <a className="ex-button" href={backHref}>Cancel</a>
              <button type="submit" className="ex-button primary" disabled={busy}>
                {busy ? 'Saving…' : id ? 'Save routing' : 'Create Routing'}
              </button>
            </footer>
          </Panel>
        </form>
      )}
    </Workspace>
  );
}

export function RoutingDetail({ id }: { id: string }) {
  const user = useCurrentUser();
  const costs = Boolean(user?.permissions.includes('production.report'));
  const canWrite = Boolean(user?.permissions.includes('production.routing') && costs);
  const router = useRouter();
  const [routing, setRouting] = useState<Routing>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [dialog, setDialog] = useState<'' | 'delete' | 'deactivate'>('');

  const load = useCallback(async () => {
    try {
      setRouting(await request<Routing>(`/production-routings/${id}`));
    } catch (cause) {
      setError(message(cause));
    }
  }, [id]);
  useEffect(() => { void load(); }, [load]);

  async function act(action: 'activate' | 'deactivate' | 'delete') {
    setBusy(true);
    setError('');
    try {
      await request(`/production-routings/${id}${action === 'delete' ? '' : `/${action}`}`, {
        method: action === 'delete' ? 'DELETE' : 'POST',
        ...(action === 'delete' ? {} : { body: '{}' }),
      });
      if (action === 'delete') {
        router.push('/production-routings');
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

  const manage = canWrite;
  const editable = routing && routing.linkedOrders === 0;

  return (
    <Workspace
      title={routing ? `${routing.partNumber} routing` : 'Routing'}
      subtitle={routing ? `${routing.name} · revision ${routing.revision} · ${routing.partName}` : 'Loading routing…'}
      crumbs={[{ label: 'Routing', href: '/production-routings' }, { label: 'Routing detail' }]}
      action={
        <>
          <a className="ex-button" href="/production-routings">All routings</a>
          {routing && <RoutingState active={routing.active} />}
        </>
      }
    >
      <ErrorMessage error={error} />
      {!routing ? (
        !error && <p className="execution-empty" role="status">Loading routing…</p>
      ) : (
        <>
          <div className="execution-actions">
            {manage && editable && <a className="ex-button" href={`/production-routings/${id}/edit`}>Edit routing</a>}
            {manage && <a className="ex-button" href={`/production-routings/new?from=${id}`}>New revision</a>}
            {manage && !routing.active && (
              <button type="button" className="ex-button primary" disabled={busy} onClick={() => void act('activate')}>
                Activate routing
              </button>
            )}
            {manage && routing.active && (
              <button type="button" className="ex-button" disabled={busy} onClick={() => setDialog('deactivate')}>
                Deactivate routing
              </button>
            )}
            {manage && editable && (
              <button type="button" className="ex-button danger" disabled={busy} onClick={() => setDialog('delete')}>
                Delete routing
              </button>
            )}
          </div>

          <Metrics
            items={[
              { label: 'Operations', value: routing.steps.length, hint: 'In sequence' },
              ...(costs ? [{ label: 'Process cost per piece', value: money(routing.totalRate, routing.currency), hint: routing.currency, tone: 'accent' } as const] : []),
              { label: 'Revision', value: routing.revision, hint: routing.active ? 'Active revision' : 'Not active' },
              { label: 'Linked orders', value: routing.linkedOrders, hint: 'Orders holding this snapshot', tone: routing.linkedOrders ? 'amber' : undefined },
            ]}
          />

          <div className="execution-snapshot">
            <span><small>Output part</small><strong>{routing.partNumber}</strong>{routing.partName}</span>
            <span><small>Output type</small><strong>{routing.kind === 'FG' ? 'Finished good' : 'Process material'}</strong></span>
            <span><small>Unit</small><strong>{routing.unitCode}</strong></span>
            <span><small>Created by</small><strong>{routing.createdBy}</strong></span>
            <span><small>Last updated</small><strong>{dateTime(routing.updatedAt)}</strong></span>
          </div>

          <Panel title="Operation sequence" note="Work flows top to bottom; each operation consumes the good output of the one above.">
            <div className="execution-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th className="numeric">Seq</th>
                    <th>Operation</th>
                    <th>Input from</th>
                    {costs && <th className="numeric">Cost per piece</th>}
                  </tr>
                </thead>
                <tbody>
                  {routing.steps.map((step, index) => (
                    <tr key={step.code}>
                      <td className="numeric">{index + 1}</td>
                      <td>
                        <strong>{step.code}</strong>
                        <small>{step.name}</small>
                      </td>
                      <td>{index === 0 ? 'Raw material issue' : `WIP ${routing.steps[index - 1].code}`}</td>
                      {costs && <td className="numeric">{money(step.rate, routing.currency)}</td>}
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={3}>{costs ? 'Total process cost per piece' : 'Operation sequence'}</td>
                    {costs && <td className="numeric">{money(routing.totalRate, routing.currency)}</td>}
                  </tr>
                </tfoot>
              </table>
            </div>
          </Panel>

          <Panel title="Change history" note="Immutable audit trail for this routing revision.">
            {routing.history.length === 0 ? (
              <Empty title="No recorded changes" />
            ) : (
              <ol className="execution-history">
                {routing.history.map((entry, index) => (
                  <li key={index}>
                    <strong>{entry.action.replaceAll('_', ' ')}</strong>
                    <small>{entry.actor} · {dateTime(entry.occurredAt)}</small>
                  </li>
                ))}
              </ol>
            )}
          </Panel>

          {dialog === 'deactivate' && (
            <ConfirmDialog
              title="Deactivate routing"
              confirmLabel="Confirm deactivation"
              cancelLabel="Keep active"
              busy={busy}
              onConfirm={() => void act('deactivate')}
              onClose={() => setDialog('')}
            >
              <p>Without an active routing, no new production order can be released for {routing.partNumber}.</p>
            </ConfirmDialog>
          )}
          {dialog === 'delete' && (
            <ConfirmDialog
              title="Delete routing"
              confirmLabel="Confirm deletion"
              cancelLabel="Keep routing"
              tone="danger"
              busy={busy}
              onConfirm={() => void act('delete')}
              onClose={() => setDialog('')}
            >
              <p>Delete revision {routing.revision} of {routing.partNumber}? Only routings without production orders can be deleted.</p>
            </ConfirmDialog>
          )}
        </>
      )}
    </Workspace>
  );
}
