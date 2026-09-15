'use client';
import { useState } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';
import { Plan, planningRequest } from './planning-api';
import { ConfirmDialog, ErrorMessage } from './execution-ui';

const labels: Record<string, string> = {
  approve: 'Approve Planning',
  close: 'Close Planning',
  orders: 'Create Production Order',
  delete: 'Delete',
};

const prompts: Record<string, string> = {
  approve: 'Approve this planning and lock its targets? Approved targets can no longer be edited.',
  close: 'Close this planning as read-only history?',
  orders: 'Create one Production Order for each planning line?',
  delete: 'Delete this draft permanently?',
};

export function PlanningActions({ plan, onChange }: {
  plan: Pick<Plan, 'id' | 'status' | 'linkedProductionOrders'>;
  onChange: (action: string) => void;
}) {
  const user = useCurrentUser();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [confirm, setConfirm] = useState('');
  const manage = user?.permissions.includes('production.plan');
  const order = user?.permissions.includes('production.order');
  const unlinked = plan.linkedProductionOrders === 0;

  async function act() {
    setPending(true);
    setError('');
    try {
      await planningRequest(`/${plan.id}${confirm === 'delete' ? '' : `/${confirm}`}`, {
        method: confirm === 'delete' ? 'DELETE' : 'POST',
      });
      const action = confirm;
      setConfirm('');
      onChange(action);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Action failed');
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="planning-actions">
      <div className="execution-actions">
        {plan.status === 'DRAFT' && manage && (
          <>
            {unlinked && (
              <>
                <a className="ex-button" href={`/production-planning/${plan.id}/edit`}>Edit</a>
                <button type="button" className="ex-button danger" disabled={pending} onClick={() => setConfirm('delete')}>
                  Delete
                </button>
              </>
            )}
            <button type="button" className="ex-button primary" disabled={pending} onClick={() => setConfirm('approve')}>
              Approve Planning
            </button>
          </>
        )}
        {plan.status === 'APPROVED' && (
          <>
            {order && unlinked && (
              <button type="button" className="ex-button primary" disabled={pending} onClick={() => setConfirm('orders')}>
                Create Production Order
              </button>
            )}
            {manage && (
              <button type="button" className="ex-button" disabled={pending} onClick={() => setConfirm('close')}>
                Close Planning
              </button>
            )}
          </>
        )}
      </div>
      {confirm && (
        <ConfirmDialog
          title={labels[confirm]}
          confirmLabel={`Confirm ${labels[confirm]}`}
          tone={confirm === 'delete' ? 'danger' : 'primary'}
          busy={pending}
          onConfirm={() => void act()}
          onClose={() => setConfirm('')}
        >
          <p>{prompts[confirm]}</p>
        </ConfirmDialog>
      )}
      <ErrorMessage error={error} />
    </div>
  );
}
