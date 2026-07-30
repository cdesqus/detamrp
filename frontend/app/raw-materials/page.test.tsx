import { describe, expect, it, vi } from 'vitest';
import { loadRawMaterialOptions } from './page';

describe('Raw Material master options', () => {
  it('loads active Supplier Unit Category and Packing options', async () => {
    const paths: string[] = [];
    vi.stubGlobal('fetch', vi.fn((path: string) => {
      paths.push(path);
      return Promise.resolve({ ok: true, json: async () => ({ items: [] }) } as Response);
    }));

    const result = await loadRawMaterialOptions(new AbortController().signal);

    expect(paths).toEqual([
      '/api/master-data/suppliers?active=true&limit=200',
      '/api/master-data/units?active=true&limit=200',
      '/api/master-data/categories?active=true&limit=200',
      '/api/master-data/packings?active=true&limit=200'
    ]);
    expect(result).toMatchObject({ supplierId: [], baseUnitId: [], categoryId: [], packingId: [] });
  });
});
