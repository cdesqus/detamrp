export type DashboardTotals = {
  plannedQty: string;
  processedQty: string;
  goodQty: string;
  rejectQty: string;
  rejectRate: string;
  achievement: string;
  wipQty: string;
  wipValue: string;
  materialActual: string;
  processActual: string;
  totalActual: string;
  entries: number;
  activeOrders: number;
  openOrders: number;
  completedOrders: number;
  currency: string;
};

export type PartPerformance = {
  partNumber: string;
  partName: string;
  unitCode: string;
  orders: number;
  plannedQty: string;
  goodQty: string;
  rejectQty: string;
  achievement: string;
};

export type ProcessWIP = { code: string; currency?: string; quantity: string; value: string };

export type OperationQuality = {
  code: string;
  processedQty: string;
  goodQty: string;
  rejectQty: string;
  rejectRate: string;
};

export type PartCost = {
  partNumber: string;
  partName: string;
  orders: number;
  goodQty: string;
  materialActual: string;
  processActual: string;
  totalActual: string;
  costPerPiece: string;
  lifetimeGoodQty?: string;
  finishedCost?: string;
  currency: string;
};

export type ProductionDashboard = {
  filter: { from: string; to: string };
  totals: DashboardTotals;
  parts: PartPerformance[];
  processes: ProcessWIP[];
  quality: OperationQuality[];
  costs: PartCost[];
  generatedAt: string;
};

/** Reject rate bands used for the headline tile only; bars stay one hue. */
export const rejectTone = (rate: string | number) =>
  Number(rate) >= 5 ? 'red' : Number(rate) >= 2 ? 'amber' : 'green';
