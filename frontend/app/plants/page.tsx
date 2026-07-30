'use client';

import { AppShell } from '../../components/app-shell/app-shell';
import { MasterDataCrud } from '../../components/master-data-crud';

export default function Page() {
  return <AppShell title="Plants"><MasterDataCrud
    title="Plants"
    description="Tujuan pengiriman yang dapat dipilih untuk satu Purchase Order."
    endpoint="/master-data/plants"
    singular="plant"
    searchPlaceholder="Search plant code or name"
    initial={{ code: '', name: '', address: '', active: true }}
    columns={[
      { key: 'code', label: 'Code' },
      { key: 'name', label: 'Name' },
      { key: 'address', label: 'Address' },
      { key: 'active', label: 'Status', render: row => row.active ? 'Active' : 'Inactive' },
      { key: 'createdByName', label: 'Created By' }
    ]}
    fields={[
      { key: 'code', label: 'Code', required: true },
      { key: 'name', label: 'Name', required: true },
      { key: 'address', label: 'Address', type: 'textarea' },
      { key: 'active', label: 'Active', type: 'checkbox' }
    ]}
  /></AppShell>;
}
