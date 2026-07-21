import { ReactNode } from 'react';

export function DashboardChart({ title, subtitle, empty, variant = 'grid', children }: { title: string; subtitle?: string; empty: boolean; variant?: 'grid' | 'donut' | 'bars'; children: ReactNode }) {
  return (
    <article className="dashboard-panel">
      <div className="panel-heading"><div><h2>{title}</h2>{subtitle ? <p>{subtitle}</p> : null}</div></div>
      <div className={`chart-stage chart-${variant}`}>
        {empty ? <div className="chart-empty"><span className="chart-empty-mark" aria-hidden="true" /><strong>Belum ada data transaksi</strong><span>Chart akan terisi otomatis dari aktivitas operasional.</span></div> : children}
      </div>
    </article>
  );
}
