'use client';

import { AppShell } from '../../components/app-shell/app-shell';
import { MasterDataCrud } from '../../components/master-data-crud';

export default function Page() {
  return <AppShell title="Categories"><MasterDataCrud
    title="Categories"
    description="Kelompok material untuk klasifikasi pembelian dan dokumen."
    endpoint="/master-data/categories"
    singular="category"
    searchPlaceholder="Search category code or name"
    initial={{ code: '', name: '', description: '', active: true }}
    columns={[
      { key: 'code', label: 'Code' },
      { key: 'name', label: 'Name' },
      { key: 'description', label: 'Description' },
      { key: 'active', label: 'Status', render: row => row.active ? 'Active' : 'Inactive' },
      { key: 'createdByName', label: 'Created By' }
    ]}
    fields={[
      { key: 'code', label: 'Code', required: true },
      { key: 'name', label: 'Name', required: true },
      { key: 'description', label: 'Description', type: 'textarea' },
      { key: 'active', label: 'Active', type: 'checkbox' }
    ]}
  /></AppShell>;
}
