import { AppShell } from '../../components/app-shell/app-shell';
import { CostIndex } from '../../components/production/cost-pages';
export default function Page() {
  return (
    <AppShell title="Production Cost">
      <CostIndex />
    </AppShell>
  );
}
