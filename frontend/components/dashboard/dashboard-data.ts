export type DashboardSnapshot = {
  filter: { from: string; to: string; supplierId?: string };
  metrics: { pendingApproval: number; openPO: number; receivedKanban: number; outstandingKanban: number; currentStock: number };
  trend: { date: string; ordered: number; received: number }[];
  poStatus: { status: string; count: number }[];
  outstandingBySupplier: { supplier: string; kanban: number }[];
  activities: { id: string; type: string; label: string; occurredAt: string }[];
};

export const emptyDashboardSnapshot: DashboardSnapshot = {
  filter: { from: '', to: '' },
  metrics: { pendingApproval: 0, openPO: 0, receivedKanban: 0, outstandingKanban: 0, currentStock: 0 },
  trend: [], poStatus: [], outstandingBySupplier: [], activities: []
};

export function defaultDashboardDates(now = new Date()) {
  const to = new Date(now), from = new Date(now);
  from.setDate(from.getDate() - 29);
  const format = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
  return { from: format(from), to: format(to) };
}
