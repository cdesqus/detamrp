import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function SuppliersPage() { return <AppShell title="Suppliers"><ModuleIndex title="Suppliers" description="Master supplier yang diselaraskan dengan directory Sage X3." actionLabel="New supplier" columns={['Supplier ID','Supplier Name','Email','Phone','Currency','Status','Created By']} searchPlaceholder="Search supplier ID or name" emptyMessage="Belum ada supplier yang dicatat." /></AppShell>; }
