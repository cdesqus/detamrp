import { AppShell } from "../../../../components/app-shell/app-shell";
import { DailyProductionForm } from "../../../../components/production/daily-production-form";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <AppShell title="Edit Daily Entry">
      <DailyProductionForm id={id} />
    </AppShell>
  );
}
