'use client';
import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useCurrentUser } from '../app-shell/app-shell';
import { Plan, period, planningRequest, quantity } from './planning-api';
import { PlanningActions } from './planning-actions';
import { Badge, Empty, ErrorMessage, Metrics, Panel, Workspace, dateTime } from './execution-ui';

export function PlanningDetail({ id }: { id: string }) {
  const router = useRouter();
  const user = useCurrentUser();
  const [plan, setPlan] = useState<Plan | null>(null);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      setPlan(await planningRequest<Plan>(`/${id}`));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Planning could not be loaded');
    }
  }, [id]);
  useEffect(() => { void load(); }, [load]);

  if (!plan) {
    return (
      <Workspace
        title="Production Planning"
        subtitle="Loading planning details…"
        crumbs={[{ label: 'Production planning', href: '/production-planning' }, { label: 'Planning detail' }]}
        action={<a className="ex-button" href="/production-planning">All planning</a>}
      >
        <ErrorMessage error={error} />
        {!error && <p className="execution-empty" role="status">Loading planning…</p>}
      </Workspace>
    );
  }

  const orderedQty = plan.orders.reduce((sum, order) => sum + Number(order.plannedQty || 0), 0);
  const goodQty = plan.orders.reduce((sum, order) => sum + Number(order.goodQty || 0), 0);

  return (
    <Workspace
      title={plan.planNumber}
      subtitle={`${plan.plantName} · ${period(plan)}`}
      crumbs={[{ label: 'Production planning', href: '/production-planning' }, { label: 'Planning detail' }]}
      action={
        <>
          <a className="ex-button" href="/production-planning">All planning</a>
          <Badge status={plan.status} />
        </>
      }
    >
      <ErrorMessage error={error} />
      <div className="execution-actions">
        <PlanningActions
          plan={plan}
          onChange={(action) => {
            if (action === 'delete') router.push('/production-planning');
            else void load();
          }}
        />
        {plan.status === 'APPROVED' && user?.permissions.includes('production.order') && (
          <a className="ex-button" href={`/production-orders/new?planId=${plan.id}`}>Release partial order</a>
        )}
      </div>

      <Metrics
        items={[
          { label: 'Planned target', value: quantity(plan.totalPlannedQty), hint: `${plan.totalPart} parts` },
          { label: 'Released to orders', value: quantity(orderedQty), hint: `${plan.linkedProductionOrders} production orders`, tone: 'accent' },
          { label: 'Actual good', value: quantity(goodQty), hint: 'Final operation output', tone: 'green' },
          { label: 'Open balance', value: quantity(Math.max(0, Number(plan.totalPlannedQty) - orderedQty)), hint: 'Not yet released', tone: 'amber' },
        ]}
      />

      <div className="execution-snapshot">
        <span><small>Period</small><strong>{period(plan)}</strong></span>
        <span><small>Plant</small><strong>{plan.plantName}</strong></span>
        <span><small>Created by</small><strong>{plan.createdBy}</strong></span>
        <span><small>Updated at</small><strong>{dateTime(plan.updatedAt)}</strong></span>
      </div>

      <Panel
        title="Planning lines"
        note="Quantities are stated in each part's own unit."
        action={<span>Total target: {quantity(plan.totalPlannedQty)}</span>}
      >
        <div className="execution-table-wrap">
          <table>
            <thead>
              <tr>
                <th>Part number</th>
                <th>Part name</th>
                <th>Unit</th>
                <th className="numeric">Target quantity</th>
              </tr>
            </thead>
            <tbody>
              {plan.lines.map((line) => (
                <tr key={line.id}>
                  <td><strong>{line.partNumber}</strong></td>
                  <td>{line.partName}</td>
                  <td>{line.unitCode}</td>
                  <td className="numeric">{quantity(line.plannedQty)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Production order progress" note="Orders released from this planning and their good output.">
        {plan.orders.length === 0 ? (
          <Empty title="No Production Orders linked yet" hint="Approve the planning, then release its lines to the floor." />
        ) : (
          <div className="execution-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Production Order</th>
                  <th>Status</th>
                  <th className="numeric">Planned quantity</th>
                  <th className="numeric">Actual good quantity</th>
                  <th>Progress</th>
                </tr>
              </thead>
              <tbody>
                {plan.orders.map((order) => {
                  const share = Number(order.plannedQty) > 0
                    ? (Number(order.goodQty) / Number(order.plannedQty)) * 100
                    : 0;
                  return (
                    <tr key={order.id}>
                      <td><a href={`/production-orders/${order.id}`}>{order.orderNumber}</a></td>
                      <td><Badge status={order.status} /></td>
                      <td className="numeric">{quantity(order.plannedQty)}</td>
                      <td className="numeric">{quantity(order.goodQty)}</td>
                      <td>
                        <div className="execution-progress">
                          <progress
                            aria-label={`Progress ${order.orderNumber}`}
                            max={100}
                            value={Math.min(100, share)}
                          />
                          <span>{quantity(share)}%</span>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Panel>

      {plan.notes && (
        <Panel title="Planning notes">
          <div className="execution-panel-body">
            <p style={{ whiteSpace: 'pre-wrap' }}>{plan.notes}</p>
          </div>
        </Panel>
      )}

      <Panel title="Change history" note="Immutable audit trail for this planning record.">
        {plan.history.length === 0 ? (
          <Empty title="No recorded changes" />
        ) : (
          <ol className="execution-history">
            {plan.history.map((entry, index) => (
              <li key={index}>
                <strong>{entry.action}</strong>
                <small>{entry.actor} · {dateTime(entry.occurredAt)}</small>
              </li>
            ))}
          </ol>
        )}
      </Panel>
    </Workspace>
  );
}
