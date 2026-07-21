import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function SupplierOrdersPage() { return <AppShell title="Supplier Orders"><ModuleIndex title="Supplier Orders" description="Kelola purchase order per supplier dan proses approval." actionLabel="Create order" columns={['PO Number','Supplier','Order Date','Expected Date','Status','Sage Number','Created By']} searchPlaceholder="Search PO or supplier" emptyMessage="Belum ada supplier order yang dibuat." /></AppShell>; }
