import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function ApprovalsPage() { return <AppShell title="Approval Inbox"><ModuleIndex title="Approval Inbox" description="Review dan approve supplier order dari dalam aplikasi." actionLabel="Open next approval" columns={['PO Number','Supplier','Order Date','Kanban','Total Value','Requested By','Status']} searchPlaceholder="Search approval" emptyMessage="Tidak ada order yang menunggu approval." /></AppShell>; }
