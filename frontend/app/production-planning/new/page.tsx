import { AppShell } from "../../../components/app-shell/app-shell";
import { PlanningForm } from "../../../components/production/planning-form";
export default function Page() {
  return (
    <AppShell title="New Planning">
      <PlanningForm />
    </AppShell>
  );
}
