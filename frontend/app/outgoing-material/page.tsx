import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function OutgoingMaterialPage() { return <AppShell title="Outgoing Material"><ModuleIndex title="Outgoing Material" description="Catat konsumsi raw material dari gudang menuju produksi." actionLabel="Create outgoing" columns={['Document Number','Date','Destination','Raw Material','Quantity','Kanban ID','Created By']} searchPlaceholder="Search document or material" emptyMessage="Belum ada material yang dikeluarkan." /></AppShell>; }
