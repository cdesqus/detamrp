import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function RawMaterialsPage() { return <AppShell title="Raw Materials"><ModuleIndex title="Raw Materials" description="Definisi material, supplier utama, base unit, dan isi per Kanban." actionLabel="New raw material" columns={['Item Code','Raw Material','Primary Supplier','Base Unit','Qty / Kanban','Currency','Unit Price','Status','Created By']} searchPlaceholder="Search item code or material" emptyMessage="Belum ada raw material. Modul ini akan diisi pada tahap CRUD berikutnya." /></AppShell>; }
