'use client';

import { AppShell } from '../../components/app-shell/app-shell';
import { MasterDataCrud } from '../../components/master-data-crud';

export default function Page() {
  return <AppShell title="Units"><MasterDataCrud
    title="Units"
    description="Unit dasar untuk accounting dan stok fisik. Kanban bukan unit dasar."
    endpoint="/master-data/units"
    singular="unit"
    searchPlaceholder="Search unit code or name"
    initial={{ code: '', name: '', decimalAllowed: false, active: true }}
    columns={[
      { key: 'code', label: 'Code' },
      { key: 'name', label: 'Name' },
      { key: 'decimalAllowed', label: 'Decimal', render: row => row.decimalAllowed ? 'Yes' : 'No' },
      { key: 'active', label: 'Status', render: row => row.active ? 'Active' : 'Inactive' },
      { key: 'createdByName', label: 'Created By' }
    ]}
    fields={[
      { key: 'code', label: 'Code', required: true },
      { key: 'name', label: 'Name', required: true },
      { key: 'decimalAllowed', label: 'Allow Decimal', type: 'checkbox' },
      { key: 'active', label: 'Active', type: 'checkbox' }
    ]}
  /></AppShell>;
}
