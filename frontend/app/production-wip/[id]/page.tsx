import { AppShell } from '../../../components/app-shell/app-shell';
import { WIPDetail } from '../../../components/production/wip-pages';
export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <AppShell title="Work in Progress">
      <WIPDetail id={id} />
    </AppShell>
  );
}
