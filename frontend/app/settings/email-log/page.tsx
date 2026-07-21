import { AppShell } from '../../../components/app-shell/app-shell';
import { ModuleIndex } from '../../../components/module-index';

export default function EmailLogPage() { return <AppShell title="Email Log"><ModuleIndex title="Email Log" description="Riwayat email approval, PO, DN, Kanban, dan hasil delivery." actionLabel="Test email" columns={['Sent At','Recipient','Email Type','Reference','Subject','Status','Error','Created By']} searchPlaceholder="Search recipient or reference" emptyMessage="Belum ada email yang dikirim dari sistem." /></AppShell>; }
