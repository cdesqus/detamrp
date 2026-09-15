import type { FormEvent, ReactNode } from 'react';
import { Icon } from '../icons';
import './execution.css';

export type Crumb = { label: string; href?: string };

/** Page frame shared by every production screen: eyebrow trail, title block and action cluster. */
export function Workspace({ title, subtitle, crumbs = [], action, children }: {
  title: string;
  subtitle: string;
  crumbs?: Crumb[];
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="execution-workspace">
      <header className="execution-heading">
        <div>
          <span className="execution-eyebrow">
            <Icon name="factory" />
            {[{ label: 'Production control' }, ...crumbs].map((crumb, index) => (
              <span key={crumb.label}>
                {index > 0 && <span className="execution-crumb-divider"> / </span>}
                {crumb.href ? <a href={crumb.href}>{crumb.label}</a> : crumb.label}
              </span>
            ))}
          </span>
          <h1>{title}</h1>
          <p>{subtitle}</p>
        </div>
        <div className="execution-actions">{action}</div>
      </header>
      {children}
    </section>
  );
}

export type Metric = { label: string; value: ReactNode; hint?: string; tone?: 'accent' | 'amber' | 'green' | 'red' };

export function Metrics({ items }: { items: Metric[] }) {
  return (
    <div className="execution-metrics">
      {items.map((item) => (
        <div className="execution-metric" key={item.label} data-tone={item.tone}>
          <span>{item.label}</span>
          <strong>{item.value}</strong>
          {item.hint && <small>{item.hint}</small>}
        </div>
      ))}
    </div>
  );
}

export function Panel({ title, note, action, children }: {
  title: string;
  note?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="execution-panel">
      <header>
        <div>
          <h2>{title}</h2>
          {note && <p>{note}</p>}
        </div>
        {action && <div className="execution-actions">{action}</div>}
      </header>
      {children}
    </section>
  );
}

export function Badge({ status }: { status: string }) {
  return <span className={`execution-badge status-${status.toLowerCase()}`}>{status.replaceAll('_', ' ')}</span>;
}

export function Tabs({ tabs, active, onSelect, label }: {
  tabs: string[];
  active: string;
  onSelect: (tab: string) => void;
  label: string;
}) {
  return (
    <div className="execution-tabs" role="tablist" aria-label={label}>
      {tabs.map((tab) => (
        <button key={tab} type="button" role="tab" aria-selected={active === tab} onClick={() => onSelect(tab)}>
          {tab}
        </button>
      ))}
    </div>
  );
}

export function Progress({ value, label, ariaLabel }: { value: number; label?: string; ariaLabel?: string }) {
  const share = Math.max(0, Math.min(100, Number.isFinite(value) ? value : 0));
  return (
    <div className="execution-progress">
      <progress max={100} value={share} aria-label={ariaLabel ?? label ?? 'Progress'} />
      <span>{label ?? `${share.toFixed(1)}%`}</span>
    </div>
  );
}

export function ErrorMessage({ error }: { error: string }) {
  return error ? <div className="execution-error" role="alert">{error}</div> : null;
}

export function Empty({ title, hint }: { title: string; hint?: string }) {
  return (
    <p className="execution-empty">
      <strong>{title}</strong>
      {hint && <span>{hint}</span>}
    </p>
  );
}

/** Modal confirmation used for irreversible order actions. */
export function ConfirmDialog({ title, confirmLabel, cancelLabel = 'Cancel', tone = 'primary', busy, onConfirm, onClose, children }: {
  title: string;
  confirmLabel: string;
  cancelLabel?: string;
  tone?: 'primary' | 'danger';
  busy?: boolean;
  onConfirm: () => void;
  onClose: () => void;
  children: ReactNode;
}) {
  function submit(event: FormEvent) {
    event.preventDefault();
    onConfirm();
  }
  return (
    <>
      <div className="execution-scrim" onClick={onClose} />
      <div className="execution-dialog" role="dialog" aria-modal="true" aria-label={title}>
        <header><h2>{title}</h2></header>
        <form onSubmit={submit}>
          <div className="execution-dialog-body">{children}</div>
          <footer className="execution-form-footer">
            <button type="button" className="ex-button" onClick={onClose} disabled={busy}>{cancelLabel}</button>
            <button type="submit" className={`ex-button ${tone === 'danger' ? 'danger primary' : 'primary'}`} disabled={busy}>
              {busy ? 'Working…' : confirmLabel}
            </button>
          </footer>
        </form>
      </div>
    </>
  );
}

export function date(value?: string) {
  if (!value) return '—';
  const parsed = new Date(value.length === 10 ? `${value}T00:00:00` : value);
  return Number.isNaN(parsed.getTime())
    ? '—'
    : parsed.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' });
}

export function dateTime(value?: string) {
  if (!value) return '—';
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? '—' : parsed.toLocaleString();
}

export const percent = (part: string | number, total: string | number) =>
  Number(total) > 0 ? (Number(part) / Number(total)) * 100 : 0;
