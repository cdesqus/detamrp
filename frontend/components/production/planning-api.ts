export type PlanLine = {
  id: string;
  finishedGoodId: string;
  rawMaterialId: string;
  partNumber: string;
  partName: string;
  unitCode: string;
  plannedQty: string;
};
export type Plan = {
  id: string;
  planNumber: string;
  periodStart: string;
  periodEnd: string;
  plantId: string;
  plantName: string;
  notes: string;
  status: "DRAFT" | "APPROVED" | "CLOSED";
  createdBy: string;
  updatedAt: string;
  totalPart: number;
  totalPlannedQty: string;
  linkedProductionOrders: number;
  lines: PlanLine[];
  orders: {
    id: string;
    orderNumber: string;
    status: string;
    plannedQty: string;
    goodQty: string;
  }[];
  history: { action: string; actor: string; occurredAt: string }[];
};
export type PartOption = {
  id: string;
  kind: string;
  partNumber: string;
  partName: string;
  unitCode: string;
};
export type PlanOptions = {
  parts: PartOption[];
  plants: { id: string; name: string }[];
};
export async function planningRequest<T>(
  path = "",
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(`/api/production-plans${path}`, {
    credentials: "include",
    ...options,
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.message ?? "Planning could not be processed");
  }
  return response.status === 204 ? (undefined as T) : response.json();
}
export function quantity(value: string | number) {
  return Number(value).toLocaleString(undefined, { maximumFractionDigits: 6 });
}
export function period(plan: Pick<Plan, "periodStart" | "periodEnd">) {
  return `${plan.periodStart.slice(0, 10)} — ${plan.periodEnd.slice(0, 10)}`;
}
