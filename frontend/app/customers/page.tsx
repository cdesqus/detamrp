'use client';

import { AppShell } from '../../components/app-shell/app-shell';
import { Customers } from '../../components/sales-master/customers';

export default function Page() {
  return <AppShell title="Customers"><Customers /></AppShell>;
}
