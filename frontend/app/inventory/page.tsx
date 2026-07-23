import { AppShell } from '../../components/app-shell/app-shell';
import { InventoryIndex } from '../../components/inventory/inventory-index';

export default function InventoryPage() {
  return <AppShell title="Stock Inventory"><InventoryIndex /></AppShell>;
}
