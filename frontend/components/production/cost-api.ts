export type OperationCost = {
  operationId: string;
  code: string;
  name: string;
  sequence: number;
  final: boolean;
  processedQty: string;
  goodQty: string;
  rejectQty: string;
  rateSnapshot: string;
  estimateCost: string;
  actualCost: string;
  costPerPiece: string;
  entries: number;
};

export type MaterialCost = {
  materialId: string;
  partNumber: string;
  partName: string;
  unitCode: string;
  usedQty: string;
  standardQty: string;
  qtyVariance: string;
  averagePrice: string;
  actualCost: string;
};

export type OrderCost = {
  orderId: string;
  orderNumber: string;
  planNumber: string;
  partNumber: string;
  partName: string;
  unitCode: string;
  plantName: string;
  status: string;
  currency: string;
  periodStart: string;
  dueDate: string;
  plannedQty: string;
  goodQty: string;
  rejectQty: string;
  materialEstimate: string;
  processEstimate: string;
  totalEstimate: string;
  materialActual: string;
  processActual: string;
  totalActual: string;
  wipValue: string;
  finishedCost: string;
  plannedUnitCost: string;
  actualUnitCost: string;
  variance: string;
  variancePercent: string;
  entryCount: number;
  updatedAt: string;
  operations: OperationCost[];
  materials: MaterialCost[];
};

export type PeriodCost = {
  period: string;
  closed: boolean;
  closedBy: string;
  orders: number;
  entries: number;
  processedQty: string;
  goodQty: string;
  rejectQty: string;
  materialActual: string;
  processActual: string;
  totalActual: string;
  currency: string;
};

/** Over the release estimate reads as a warning, at or below it as good news. */
export const varianceTone = (value: string | number) =>
  Number(value) > 0 ? 'cancelled' : Number(value) < 0 ? 'completed' : 'released';
