import { AppShell } from "../../../components/app-shell/app-shell";
import { DailyProductionForm } from "../../../components/production/daily-production-form";
export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ orderId?: string }>;
}) {
  const { orderId } = await searchParams;
  return (
    <AppShell title="New Daily Entry">
      <DailyProductionForm orderId={orderId} />
    </AppShell>
  );
}
