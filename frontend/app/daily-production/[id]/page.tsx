import { AppShell } from "../../../components/app-shell/app-shell";
import { DailyProductionDetail } from "../../../components/production/daily-production-pages";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <AppShell title="Daily Production">
      <DailyProductionDetail id={id} />
    </AppShell>
  );
}
