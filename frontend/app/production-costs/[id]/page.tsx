import { AppShell } from '../../../components/app-shell/app-shell';
import { CostDetail } from '../../../components/production/cost-pages';
export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <AppShell title="Production Cost">
      <CostDetail id={id} />
    </AppShell>
  );
}
