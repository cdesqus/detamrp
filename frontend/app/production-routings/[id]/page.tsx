import { AppShell } from '../../../components/app-shell/app-shell';
import { RoutingDetail } from '../../../components/production/routing-pages';
export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <AppShell title="Routing">
      <RoutingDetail id={id} />
    </AppShell>
  );
}
