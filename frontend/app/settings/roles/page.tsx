import { AppShell } from '../../../components/app-shell/app-shell';
import { ModuleIndex } from '../../../components/module-index';

export default function RolesPage() { return <AppShell title="Roles & Permissions"><ModuleIndex title="Roles & Permissions" description="Atur role dan permission untuk setiap fungsi operasional." actionLabel="New role" columns={['Role Code','Role Name','Users','Permissions','Status','Created By']} searchPlaceholder="Search role" emptyMessage="Role bawaan sistem akan tampil setelah CRUD RBAC diaktifkan." /></AppShell>; }
