import { AppShell } from '../../components/app-shell/app-shell';
import { OrderIndex } from '../../components/production/order-pages';
export default function Page() {
  return (
    <AppShell title="Production Orders">
      <OrderIndex />
    </AppShell>
  );
}
