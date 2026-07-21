export type DashboardSnapshot = {
  metrics: { pendingApproval: number; expectedToday: number; receivedToday: number; outstandingKanban: number };
  trend: { date: string; ordered: number; received: number }[];
  poStatus: { status: string; count: number }[];
  outstandingBySupplier: { supplier: string; kanban: number }[];
  activities: { id: string; label: string; occurredAt: string }[];
};

export const emptyDashboardSnapshot: DashboardSnapshot = {
  metrics: { pendingApproval: 0, expectedToday: 0, receivedToday: 0, outstandingKanban: 0 },
  trend: [], poStatus: [], outstandingBySupplier: [], activities: []
};
