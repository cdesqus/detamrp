import { AppShell } from "../../components/app-shell/app-shell";
import { PlanningIndex } from "../../components/production/planning-index";
export default function Page() {
  return (
    <AppShell title="Production Planning">
      <PlanningIndex />
    </AppShell>
  );
}
