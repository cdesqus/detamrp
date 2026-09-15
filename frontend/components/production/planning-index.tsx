'use client';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';
import { Plan, period, planningRequest, quantity } from './planning-api';
import { PlanningActions } from './planning-actions';
import { Badge, Empty, ErrorMessage, Metrics, Panel, Workspace, dateTime } from './execution-ui';

const PAGE_SIZE = 20;
const PLAN_STATUSES = ['DRAFT', 'APPROVED', 'CLOSED'];

export function PlanningIndex() {
  const user = useCurrentUser();
  const [items, setItems] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const result = await planningRequest<{ items: Plan[] }>();
      setItems(result.items ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Planning could not be loaded');
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { void load(); }, [load]);

  const filtered = useMemo(() => items.filter((plan) =>
    (!status || plan.status === status) &&
    `${plan.planNumber} ${plan.plantName}`.toLowerCase().includes(search.toLowerCase())), [items, status, search]);
  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const current = Math.min(page, pages);
  const visible = filtered.slice((current - 1) * PAGE_SIZE, current * PAGE_SIZE);
  const count = (value: string) => items.filter((plan) => plan.status === value).length;
  const plannedTotal = items.reduce((sum, plan) => sum + Number(plan.totalPlannedQty || 0), 0);

  return (
    <Workspace
      title="Production Planning"
      subtitle="Monthly production targets per plant and the production orders released from them."
      crumbs={[{ label: 'Production planning' }]}
      action={user?.permissions.includes('production.plan') && (
        <a className="ex-button primary" href="/production-planning/new">New Planning</a>
      )}
    >
      <Metrics
        items={[
          { label: 'Planning records', value: items.length, hint: 'All periods' },
          { label: 'Draft', value: count('DRAFT'), hint: 'Editable targets', tone: 'amber' },
          { label: 'Approved', value: count('APPROVED'), hint: 'Ready to release', tone: 'accent' },
          { label: 'Planned quantity', value: quantity(plannedTotal), hint: 'Across all plans' },
        ]}
      />
      <ErrorMessage error={error} />
      <Panel
        title="Planning register"
        note="Approved planning is locked; targets can only be released into production orders."
        action={<span>{filtered.length} records</span>}
      >
        <div className="execution-filters">
          <label>
            Search
            <input
              type="search"
              placeholder="Planning number or plant"
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
              {PLAN_STATUSES.map((value) => <option key={value} value={value}>{value}</option>)}
            </select>
          </label>
          <button type="button" className="ex-button" onClick={() => void load()} disabled={loading}>Refresh</button>
        </div>
        {loading ? (
          <p className="execution-empty" role="status">Loading planning…</p>
        ) : visible.length === 0 ? (
          <Empty
            title={error ? 'Planning unavailable' : 'No planning found'}
            hint="Create a planning or adjust the filters."
          />
        ) : (
          <div className="execution-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Planning Number</th>
                  <th>Period</th>
                  <th>Plant</th>
                  <th className="numeric">Total Part</th>
                  <th className="numeric">Total Planned Qty</th>
                  <th className="numeric">Linked Production Orders</th>
                  <th>Status</th>
                  <th>Updated At</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {visible.map((plan) => (
                  <tr key={plan.id}>
                    <td><a href={`/production-planning/${plan.id}`}>{plan.planNumber}</a></td>
                    <td>{period(plan)}</td>
                    <td>{plan.plantName}</td>
                    <td className="numeric">{plan.totalPart}</td>
                    <td className="numeric">{quantity(plan.totalPlannedQty)}</td>
                    <td className="numeric">{plan.linkedProductionOrders}</td>
                    <td><Badge status={plan.status} /></td>
                    <td>{dateTime(plan.updatedAt)}</td>
                    <td>
                      <div className="execution-actions">
                        <a className="ex-button" href={`/production-planning/${plan.id}`}>View</a>
                        <PlanningActions plan={plan} onChange={() => void load()} />
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="execution-pagination">
          <span>{filtered.length} planning records · page {current} of {pages}</span>
          <div className="execution-actions">
            <button type="button" className="ex-button" disabled={current === 1} onClick={() => setPage(current - 1)}>Previous</button>
            <button type="button" className="ex-button" disabled={current === pages} onClick={() => setPage(current + 1)}>Next</button>
          </div>
        </div>
      </Panel>
    </Workspace>
  );
}
