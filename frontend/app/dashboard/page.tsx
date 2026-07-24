import { Suspense } from 'react';
import { AppShell } from '../../components/app-shell/app-shell';
import { DashboardView } from '../../components/dashboard/dashboard-view';

export default function DashboardPage() {
  return <AppShell title="Dashboard">
    <Suspense fallback={<div className="dashboard-loading" role="status">Loading dashboard...</div>}><DashboardView /></Suspense>
  </AppShell>;
}
