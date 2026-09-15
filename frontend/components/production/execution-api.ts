import type { PlanOptions } from './planning-api';
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api${path}`, { credentials: 'include', ...init, headers: { 'Content-Type': 'application/json', ...init.headers } });
  if (!response.ok) { const body = await response.json().catch(() => ({})); throw new Error(body.message || body.error || 'The request could not be completed. Please try again.'); }
  return response.status === 204 ? undefined as T : response.json();
}
export const quantity = (value: string | number = 0) => Number(value).toLocaleString(undefined, { maximumFractionDigits: 6 });
export const money = (value: string | number = 0, currency = 'IDR') => new Intl.NumberFormat(undefined, { style: 'currency', currency, maximumFractionDigits: 2 }).format(Number(value));
export type OrderLine = { id: string; planId: string; planNumber: string; partNumber: string; partName: string; unitCode: string; plantName: string; periodStart: string; periodEnd: string; remainingQty: string };
export type OrderOptions = PlanOptions & { lines: OrderLine[] };
export type Order = { id: string; orderNumber: string; planId: string; planLineId: string; planNumber: string; partNumber: string; partName: string; unitCode: string; plantName: string; plannedQty: string; periodStart: string; dueDate: string; status: 'RELEASED'|'IN_PROGRESS'|'PARTIAL'|'COMPLETED'|'CANCELLED'; notes: string; bomRevision: number; currency: string; materialEstimate: string; processEstimate: string; actualGood: string; rejectQty: string; materialCost: string; processCost: string; wipQty: string; entryCount: number; updatedAt: string; createdBy: string; operations: {id:string;code:string;name:string;sequence:number;rate:string;plannedQty:string;processedQty:string;goodQty:string;rejectQty:string;wipQty:string;onHandQty?:string;stagedQty?:string}[]; materials: {id:string;partNumber:string;partName:string;unitCode:string;usageQty:string;unitPrice:string;currency:string}[]; history: {action:string;actor:string;occurredAt:string}[] };
export type RoutingStep = { code: string; name: string; rate: string };
export type Routing = { id: string; partId: string; kind: 'FG' | 'RAW_MATERIAL'; partNumber: string; partName: string; unitCode: string; name: string; currency: string; revision: number; active: boolean; steps: RoutingStep[]; totalRate: string; linkedOrders: number; updatedAt: string; createdBy: string; history: { action: string; actor: string; occurredAt: string }[] };
export type RoutingOptions = { parts: PlanOptions['parts'] };
