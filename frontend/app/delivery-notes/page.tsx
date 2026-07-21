import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function DeliveryNotesPage() { return <AppShell title="Delivery Notes"><ModuleIndex title="Delivery Notes" description="DN dan Kanban yang dibuat otomatis setelah PO disetujui." actionLabel="Download documents" columns={['DN Number','PO Number','Supplier','Expected Date','Kanban Lots','Status','Created By']} searchPlaceholder="Search DN, PO, or supplier" emptyMessage="Belum ada delivery note dari PO approved." /></AppShell>; }
