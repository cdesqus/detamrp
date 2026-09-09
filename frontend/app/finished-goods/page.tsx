'use client';

import { AppShell } from '../../components/app-shell/app-shell';
import { FinishedGoods } from '../../components/sales-master/finished-goods';

export default function Page() {
  return <AppShell title="Finished Goods"><FinishedGoods /></AppShell>;
}
