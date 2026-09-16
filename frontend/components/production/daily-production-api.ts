import type { Order } from "./execution-api";
export type DailyMaterial = {
  materialId: string;
  quantity: string;
  partNumber: string;
  partName: string;
  unitCode: string;
  unitPrice: string;
  cost: string;
};
export type ProductionEntry = {
  id: string;
  entryNumber: string;
  orderId: string;
  orderNumber: string;
  operationId: string;
  productionDate: string;
  shift: string;
  operatorId: string;
  operatorName: string;
  operationCode: string;
  operationName: string;
  sequence: number;
  partNumber: string;
  partName: string;
  unitCode: string;
  plantName: string;
  processed: string;
  good: string;
  rejected: string;
  status: "POSTED" | "VOIDED";
  notes: string;
  version: number;
  canCorrect: boolean;
  periodClosed: boolean;
  effectsLocked: boolean;
  createdBy: string;
  updatedAt: string;
  currency: string;
  materialCost: string;
  processCost: string;
  processRate: string;
  materials: DailyMaterial[];
  voidReason: string;
  history: { action: string; actor: string; occurredAt: string }[];
};
export type DailyOptions = {
  orders: Order[];
  operators: { id: string; name: string }[];
  closedPeriods: string[];
};
export function inputRemaining(order: Order, index: number) {
  if (index < 0) return 0;
  // Later operations work on what the previous operation has finished; the
  // system moves it across on its own.
  if (index > 0) {
    return Math.max(
      0,
      Number(order.operations[index].stagedQty ?? 0) + Number(order.operations[index - 1].onHandQty ?? 0),
    );
  }
  return Math.max(
    0,
    Number(order.plannedQty) - Number(order.operations[0].processedQty),
  );
}
export function errorText(error: unknown) {
  return error instanceof Error
    ? error.message
    : "Production could not be processed";
}
