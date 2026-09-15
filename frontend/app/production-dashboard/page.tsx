import { AppShell } from '../../components/app-shell/app-shell';
import { ProductionDashboardView } from '../../components/production/dashboard-pages';
export default function Page() {
  return (
    <AppShell title="Production Dashboard">
      <ProductionDashboardView />
    </AppShell>
  );
}
