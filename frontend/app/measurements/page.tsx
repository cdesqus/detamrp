import { AppShell } from '../../components/app-shell/app-shell';
import { ModuleIndex } from '../../components/module-index';

export default function MeasurementsPage() { return <AppShell title="Measurements"><ModuleIndex title="Measurements" description="Unit dasar untuk accounting dan stok fisik." actionLabel="New measurement" columns={['Code','Name','Precision','Status','Created By','Updated At']} searchPlaceholder="Search unit code or name" emptyMessage="Belum ada measurement tambahan." /></AppShell>; }
