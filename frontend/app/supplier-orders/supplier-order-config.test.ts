import { describe, expect, it } from 'vitest';
import { supplierOrderColumns } from './supplier-order-config';

describe('supplierOrderColumns', () => {
  it('keeps PO, DN, and Kanban documents with their supplier order', () => {
    expect(supplierOrderColumns).toEqual(expect.arrayContaining(['PO Document', 'DN Documents', 'Kanban Labels']));
  });
});
