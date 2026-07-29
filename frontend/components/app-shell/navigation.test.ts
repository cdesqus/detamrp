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
