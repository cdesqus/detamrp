import { describe, expect, it } from 'vitest';
import { formatMoney, formatQuantity } from './number-format';

describe('number formatting', () => {
  it('formats IDR without database-scale zeroes', () => {
    expect(formatMoney('10000000.000000', 'IDR')).toBe('IDR 10.000.000');
  });

  it('keeps useful non-IDR fractions', () => {
    expect(formatMoney('12.500000', 'USD')).toBe('USD 12,5');
  });

  it('trims quantity zeroes and groups digits', () => {
    expect(formatQuantity('5.000000')).toBe('5');
    expect(formatQuantity('12345.250000')).toBe('12.345,25');
  });
});
