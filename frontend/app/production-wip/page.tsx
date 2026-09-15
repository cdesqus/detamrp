import { AppShell } from '../../components/app-shell/app-shell';
import { WIPIndex } from '../../components/production/wip-pages';
export default function Page() {
  return (
    <AppShell title="Work in Progress">
      <WIPIndex />
    </AppShell>
  );
}
