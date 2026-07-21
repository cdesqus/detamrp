import { AppShell } from '../../../components/app-shell/app-shell';
import { ModuleIndex } from '../../../components/module-index';

export default function UsersPage() { return <AppShell title="Users"><ModuleIndex title="Users" description="Kelola akun username/password dan status akses aplikasi." actionLabel="New user" columns={['Username','Display Name','Roles','Status','Last Login','Created By']} searchPlaceholder="Search username or name" emptyMessage="Belum ada user tambahan selain administrator awal." /></AppShell>; }
