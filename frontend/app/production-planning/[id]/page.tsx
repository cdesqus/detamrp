import { AppShell } from "../../../components/app-shell/app-shell";
import { PlanningDetail } from "../../../components/production/planning-detail";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <AppShell title="Production Planning">
      <PlanningDetail id={id} />
    </AppShell>
  );
}
