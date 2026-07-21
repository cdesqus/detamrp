import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';
import { supplierOrderColumns } from './supplier-order-config';

export default function SupplierOrdersPage() { return <AppShell title="Supplier Orders"><ModuleIndex title="Supplier Orders" description="Kelola PO dan akses seluruh dokumen PO, DN, serta Kanban dari satu tempat." actionLabel="Create order" columns={supplierOrderColumns} searchPlaceholder="Search PO or supplier" emptyMessage="Belum ada supplier order yang dibuat." /></AppShell>; }
