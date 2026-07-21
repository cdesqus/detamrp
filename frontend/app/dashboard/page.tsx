import { AppShell } from '../../components/app-shell/app-shell';
import { DashboardChart } from '../../components/dashboard/dashboard-chart';
import { emptyDashboardSnapshot } from '../../components/dashboard/dashboard-data';

export default function DashboardPage() {
  const data = emptyDashboardSnapshot;
  const metrics = [
    { label: 'Pending Approval', value: data.metrics.pendingApproval, note: 'Waiting for director' },
    { label: 'Expected Today', value: data.metrics.expectedToday, note: 'Planned delivery' },
    { label: 'Received Today', value: data.metrics.receivedToday, note: 'Completed receiving' },
    { label: 'Outstanding Kanban', value: data.metrics.outstandingKanban, note: 'Not yet received' }
  ];
  return (
    <AppShell title="Dashboard">
      <section className="dashboard-page">
        <div className="page-title-row"><div><h1>Dashboard</h1><p className="muted">Ringkasan procurement dan warehouse hari ini.</p></div><span className="live-badge"><i /> Live data</span></div>
        <div className="metric-grid">
          {metrics.map(metric => <article key={metric.label}><span>{metric.label}</span><strong>{metric.value}</strong><small>{metric.note}</small></article>)}
        </div>
        <div className="dashboard-grid">
          <DashboardChart title="PO & Receiving Trend" subtitle="Last 30 days" empty={data.trend.length === 0}><svg aria-label="PO and receiving trend" /></DashboardChart>
          <DashboardChart title="PO Status" subtitle="Current distribution" empty={data.poStatus.length === 0} variant="donut"><svg aria-label="PO status distribution" /></DashboardChart>
          <DashboardChart title="Outstanding by Supplier" subtitle="Kanban not yet received" empty={data.outstandingBySupplier.length === 0} variant="bars"><svg aria-label="Outstanding Kanban by supplier" /></DashboardChart>
          <article className="dashboard-panel activity-panel">
            <div className="panel-heading"><div><h2>Latest Activity</h2><p>Recent operational updates</p></div></div>
            {data.activities.length === 0 ? <div className="activity-empty"><span>◎</span><strong>Belum ada aktivitas</strong><small>Aktivitas PO dan receiving akan tampil di sini.</small></div> : null}
          </article>
        </div>
      </section>
    </AppShell>
  );
}
