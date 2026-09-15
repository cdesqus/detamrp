'use client';
import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plan, PlanOptions, planningRequest, quantity } from './planning-api';
import { ErrorMessage, Panel, Workspace } from './execution-ui';

type DraftLine = { kind: string; partId: string; qty: string };
const emptyLine = (): DraftLine => ({ kind: 'FG', partId: '', qty: '' });
const NIL_ID = '00000000-0000-0000-0000-000000000000';

export function PlanningForm({ id }: { id?: string }) {
  const router = useRouter();
  const [options, setOptions] = useState<PlanOptions | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [number, setNumber] = useState('Assigned when saved');
  const [start, setStart] = useState('');
  const [end, setEnd] = useState('');
  const [plant, setPlant] = useState('');
  const [notes, setNotes] = useState('');
  const [lines, setLines] = useState<DraftLine[]>([emptyLine()]);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let active = true;
    Promise.all([
      planningRequest<PlanOptions>('/options'),
      id ? planningRequest<Plan>(`/${id}`) : Promise.resolve(null),
    ])
      .then(([loadedOptions, plan]) => {
        if (!active) return;
        if (plan) {
          if (plan.status !== 'DRAFT' || plan.linkedProductionOrders > 0) {
            throw new Error('Only an unlinked Draft planning can be edited');
          }
          setNumber(plan.planNumber);
          setStart(plan.periodStart.slice(0, 10));
          setEnd(plan.periodEnd.slice(0, 10));
          setPlant(plan.plantId);
          setNotes(plan.notes);
          setLines(plan.lines.map((line) => ({
            kind: line.finishedGoodId !== NIL_ID ? 'FG' : 'RAW_MATERIAL',
            partId: line.finishedGoodId !== NIL_ID ? line.finishedGoodId : line.rawMaterialId,
            qty: line.plannedQty,
          })));
        }
        setOptions(loadedOptions);
      })
      .catch((cause) => { if (active) setError(cause.message); })
      .finally(() => { if (active) setLoaded(true); });
    return () => { active = false; };
  }, [id]);

  function change(index: number, patch: Partial<DraftLine>) {
    setLines((current) => current.map((line, position) => (position === index ? { ...line, ...patch } : line)));
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    setError('');
    if (end < start) {
      setError('Period end cannot precede period start');
      return;
    }
    if (!lines.length || lines.some((line) => !line.partId || !Number.isFinite(Number(line.qty)) || Number(line.qty) <= 0)) {
      setError('Select a part and a positive quantity for every line');
      return;
    }
    setSaving(true);
    try {
      const plan = await planningRequest<Plan>(id ? `/${id}` : '', {
        method: id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          periodStart: `${start}T00:00:00Z`,
          periodEnd: `${end}T00:00:00Z`,
          plantId: plant,
          notes,
          lines: lines.map((line) => ({
            [line.kind === 'FG' ? 'finishedGoodId' : 'rawMaterialId']: line.partId,
            plannedQty: line.qty,
          })),
        }),
      });
      router.push(`/production-planning/${plan.id}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Planning could not be saved');
    } finally {
      setSaving(false);
    }
  }

  const backHref = id ? `/production-planning/${id}` : '/production-planning';
  const total = lines.reduce((sum, line) => sum + (Number(line.qty) || 0), 0);

  return (
    <Workspace
      title={id ? 'Edit Planning' : 'New Planning'}
      subtitle="Set monthly production targets per part number, plant and period."
      crumbs={[{ label: 'Production planning', href: '/production-planning' }, { label: id ? 'Edit' : 'New planning' }]}
      action={<a className="ex-button" href="/production-planning">Back to planning</a>}
    >
      <ErrorMessage error={error} />
      {!loaded ? (
        <p className="execution-empty" role="status">Loading planning…</p>
      ) : (
        options && (
          <form onSubmit={save}>
            <fieldset disabled={saving} className="planning-fieldset">
              <Panel title="Planning period" note="The planning number is assigned by the server when the draft is saved.">
                <div className="execution-panel-body execution-form-grid">
                  <label>
                    Planning number
                    <input readOnly value={number} />
                  </label>
                  <label>
                    Plant
                    <select required value={plant} onChange={(event) => setPlant(event.target.value)}>
                      <option value="">Select plant</option>
                      {options.plants.map((option) => <option key={option.id} value={option.id}>{option.name}</option>)}
                    </select>
                  </label>
                  <label>
                    Period start
                    <input required type="date" value={start} onChange={(event) => setStart(event.target.value)} />
                  </label>
                  <label>
                    Period end
                    <input required type="date" min={start} value={end} onChange={(event) => setEnd(event.target.value)} />
                  </label>
                  <label className="full">
                    Notes
                    <textarea rows={3} value={notes} onChange={(event) => setNotes(event.target.value)} />
                  </label>
                </div>
              </Panel>

              <Panel
                title="Planning lines"
                note="Each line targets one active part; quantities follow that part's unit."
                action={
                  <button type="button" className="ex-button" onClick={() => setLines((current) => [...current, emptyLine()])}>
                    + Add part
                  </button>
                }
              >
                <div className="execution-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Output type</th>
                        <th>Part number</th>
                        <th>Part name</th>
                        <th>Unit</th>
                        <th className="numeric">Target quantity</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {lines.map((line, index) => {
                        const part = options.parts.find((option) => option.id === line.partId && option.kind === line.kind);
                        return (
                          <tr key={index}>
                            <td>
                              <select
                                aria-label={`Output type ${index + 1}`}
                                value={line.kind}
                                onChange={(event) => change(index, { kind: event.target.value, partId: '' })}
                              >
                                <option value="FG">Finished Good</option>
                                <option value="RAW_MATERIAL">Process Material</option>
                              </select>
                            </td>
                            <td>
                              <select
                                aria-label={`Part number ${index + 1}`}
                                required
                                value={part ? line.partId : ''}
                                onChange={(event) => change(index, { partId: event.target.value })}
                              >
                                <option value="">Select active part</option>
                                {options.parts.filter((option) => option.kind === line.kind).map((option) => (
                                  <option key={option.id} value={option.id}>{option.partNumber} — {option.partName}</option>
                                ))}
                              </select>
                            </td>
                            <td>{part?.partName ?? '—'}</td>
                            <td>{part?.unitCode ?? '—'}</td>
                            <td className="numeric">
                              <input
                                aria-label={`Target quantity ${index + 1}`}
                                required
                                type="number"
                                min="0.000001"
                                max="99999999999999.999999"
                                step="0.000001"
                                value={line.qty}
                                onChange={(event) => change(index, { qty: event.target.value })}
                              />
                            </td>
                            <td>
                              <button
                                type="button"
                                className="ex-button danger"
                                aria-label={`Remove line ${index + 1}`}
                                disabled={lines.length === 1}
                                onClick={() => setLines((current) => current.filter((_, position) => position !== index))}
                              >
                                Remove
                              </button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                    <tfoot>
                      <tr>
                        <td colSpan={4}>Total target across {lines.length} lines</td>
                        <td className="numeric">{quantity(total)}</td>
                        <td />
                      </tr>
                    </tfoot>
                  </table>
                </div>
                <footer className="execution-form-footer">
                  <span className="execution-footer-note">Quantities are shown in each part’s own unit and are not summed across units.</span>
                  <a className="ex-button" href={backHref}>Cancel</a>
                  <button type="submit" className="ex-button primary" disabled={saving}>
                    {saving ? 'Saving…' : 'Save Draft'}
                  </button>
                </footer>
              </Panel>
            </fieldset>
          </form>
        )
      )}
    </Workspace>
  );
}
