import { AppShell } from "../../../../components/app-shell/app-shell";
import { PlanningForm } from "../../../../components/production/planning-form";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <AppShell title="Edit Planning">
      <PlanningForm id={id} />
    </AppShell>
  );
}
