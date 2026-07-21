import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function ReceivingPage() { return <AppShell title="Receiving"><ModuleIndex title="Receiving" description="Mulai sesi receiving dan scan Kanban berdasarkan DN." actionLabel="Create receiving" columns={['Receiving Number','DN Number','PO Number','Supplier','Received At','Kanban','Outstanding','Sage Number','Created By']} searchPlaceholder="Search receiving or DN" emptyMessage="Belum ada receiving yang diselesaikan." /></AppShell>; }
