'use client';

import { AppShell } from '../../components/app-shell/app-shell';
import { MasterDataCrud } from '../../components/master-data-crud';
import { loadRawMaterialOptions } from './raw-material-options';

export default function Page() {
  return <AppShell title="Raw Materials"><MasterDataCrud
    title="Raw Materials"
    description="Material definitions, suppliers, units, categories, packaging, standard prices, and quantity per Kanban."
    endpoint="/master-data/raw-materials"
    singular="raw material"
    searchPlaceholder="Search item code, Sage code, or material"
    onLoadFields={loadRawMaterialOptions}
    initial={{
      code: '', sageItemCode: '', name: '', supplierId: '', baseUnitId: '', categoryId: '', packingId: '',
      qtyPerKanban: 1, minimumStock: 0, standardUnitPrice: 0, description: '', active: true
    }}
    columns={[
      { key: 'code', label: 'Item Code' },
      { key: 'name', label: 'Raw Material' },
      { key: 'categoryName', label: 'Category' },
      { key: 'packingName', label: 'Packing' },
      { key: 'supplierName', label: 'Primary Supplier' },
      { key: 'baseUnitCode', label: 'Base Unit' },
      { key: 'qtyPerKanban', label: 'Qty / Kanban' },
      { key: 'standardUnitPrice', label: 'Unit Price' },
      { key: 'currency', label: 'Currency' },
      { key: 'active', label: 'Status', render: row => row.active ? 'Active' : 'Inactive' },
      { key: 'createdByName', label: 'Created By' }
    ]}
    fields={[
      { key: 'code', label: 'Item Code', required: true },
      { key: 'sageItemCode', label: 'Sage Item Code', required: true },
      { key: 'name', label: 'Raw Material Name', required: true },
      { key: 'supplierId', label: 'Primary Supplier', type: 'select', required: true },
      { key: 'baseUnitId', label: 'Base Unit', type: 'select', required: true },
      { key: 'categoryId', label: 'Category', type: 'select', required: true },
      { key: 'packingId', label: 'Packing', type: 'select', required: true },
      { key: 'qtyPerKanban', label: 'Qty per Kanban', type: 'number', step: '0.000001', required: true },
      { key: 'minimumStock', label: 'Minimum Stock', type: 'number', step: '0.000001', required: true },
      { key: 'standardUnitPrice', label: 'Standard Unit Price', type: 'number', step: '0.000001', required: true },
      { key: 'description', label: 'Description', type: 'textarea' },
      { key: 'active', label: 'Active', type: 'checkbox' }
    ]}
  /></AppShell>;
}
