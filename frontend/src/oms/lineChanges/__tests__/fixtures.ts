import type { SalesOrderLineChange } from '../../types'

export function change(overrides: Partial<SalesOrderLineChange> = {}): SalesOrderLineChange {
  return {
    id: 'c-1', step: 'packing', action: 'substitute', line_item_id: 'line-1', sku_id: 'sku-2', sku_code: 'FEP-1',
    product_name: 'Feijão Preto 1kg', variant_name: 'Default', quantity: 4, original_qty_covered: 4,
    return_to_stock: false, price_minor_units: null, reason: '', by: 'user-1', at: '2026-10-24T10:49:00Z', ...overrides,
  }
}
