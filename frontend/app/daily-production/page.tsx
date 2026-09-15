import { AppShell } from "../../components/app-shell/app-shell";
import { DailyProductionIndex } from "../../components/production/daily-production-pages";
export default function Page() {
  return (
    <AppShell title="Daily Production">
      <DailyProductionIndex />
    </AppShell>
  );
}
