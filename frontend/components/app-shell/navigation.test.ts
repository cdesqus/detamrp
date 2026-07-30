import { describe, expect, it } from 'vitest';
import { firstPermittedRoute, requiredPermissionForPath, visibleNavigationGroups } from './navigation';

describe('navigation permission policy', () => {
  it('matches specific and parent supplier-order routes', () => {
    expect(requiredPermissionForPath('/supplier-orders/new')).toBe('po.create');
    expect(requiredPermissionForPath('/supplier-orders/123')).toBe('po.view');
    expect(requiredPermissionForPath('/unknown')).toBeNull();
  });

  it('maps protected operational and settings routes', () => {
    expect(requiredPermissionForPath('/dashboard')).toBe('dashboard.view');
    expect(requiredPermissionForPath('/receiving/session-1')).toBe('receiving.view');
    expect(requiredPermissionForPath('/settings/roles')).toBe('role.manage');
    expect(requiredPermissionForPath('/settings/company')).toBe('configuration.manage');
    expect(requiredPermissionForPath('/units')).toBe('master_data.view');
    expect(requiredPermissionForPath('/categories')).toBe('master_data.view');
    expect(requiredPermissionForPath('/packings')).toBe('master_data.view');
    expect(requiredPermissionForPath('/plants')).toBe('master_data.view');
  });

  it('groups Unit Category and Packing below Measurements', () => {
    const dataMaster = visibleNavigationGroups(['master_data.view']).find(group => group.label === 'Data Master');
    expect(dataMaster?.items.map(item => item.label)).toEqual(['Measurements', 'Plants', 'Suppliers', 'Raw Materials']);
    const measurements = dataMaster?.items[0];
    expect(measurements && 'items' in measurements ? measurements.items.map(item => [item.label, item.href]) : []).toEqual([
      ['Unit', '/units'],
      ['Category', '/categories'],
      ['Packing', '/packings']
    ]);
    expect(firstPermittedRoute(['master_data.view'])).toBe('/units');
  });

  it('keeps permitted items and removes empty groups', () => {
    const groups = visibleNavigationGroups(['role.manage']);
    expect(groups.flatMap(group => group.items).map(item => item.label)).toEqual(['Roles & Permissions']);
    expect(groups.map(group => group.label).filter(Boolean)).toEqual(['Settings']);
  });

  it('selects the first permitted route deterministically', () => {
    expect(firstPermittedRoute(['inventory.view'])).toBe('/inventory');
    expect(firstPermittedRoute(['role.manage'])).toBe('/settings/roles');
    expect(firstPermittedRoute([])).toBeNull();
  });
});
