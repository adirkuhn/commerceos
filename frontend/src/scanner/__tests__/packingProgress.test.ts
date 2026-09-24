import { describe, it, expect } from 'vitest'
import { packedQuantityByLineItem, computeMissingPackItems } from '../packingProgress'
import type { PackScan, PickingSessionLineItem } from '../types'

function packScan(overrides: Partial<PackScan> = {}): PackScan {
  return {
    id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack',
    quantity: 1, scanned_at: '2026-08-22T10:00:00Z', scanned_by: 'user-1',
    ...overrides,
  }
}

describe('packedQuantityByLineItem', () => {
  it('sums pack scans by quantity, per line item', () => {
    const result = packedQuantityByLineItem([
      packScan({ id: 'scan-1', quantity: 2 }),
      packScan({ id: 'scan-2', quantity: 1 }),
    ])
    expect(result.get('li-1')).toBe(3)
  })

  it('keeps different line items separate', () => {
    const result = packedQuantityByLineItem([
      packScan({ id: 'scan-1', line_item_id: 'li-1', quantity: 2 }),
      packScan({ id: 'scan-2', line_item_id: 'li-2', quantity: 5 }),
    ])
    expect(result.get('li-1')).toBe(2)
    expect(result.get('li-2')).toBe(5)
  })
})

describe('packedQuantityByLineItem order changes', () => {
  it('counts a substitute by the original quantity it covers, not its own', () => {
    const result = packedQuantityByLineItem([packScan({ action_type: 'substitute', quantity: 8, original_quantity_covered: 2 })])
    expect(result.get('li-1')).toBe(2)
  })

  it('counts refunded units as handled', () => {
    const result = packedQuantityByLineItem([packScan({ action_type: 'refund', quantity: 1 })])
    expect(result.get('li-1')).toBe(1)
  })

  it('ignores extra items, which cover no line', () => {
    const result = packedQuantityByLineItem([packScan({ action_type: 'freebie', line_item_id: null, quantity: 3 })])
    expect(result.size).toBe(0)
  })
})

describe('computeMissingPackItems', () => {
  const lineItems: PickingSessionLineItem[] = [
    { id: 'li-1', sku_id: 'sku-1', name: 'Beans', sku_code: 'BEANS-1', barcode: '111', quantity: 4, price_minor_units: 100, total_minor_units: 400 },
  ]

  it('reports nothing missing once packed reaches the full quantity', () => {
    const missing = computeMissingPackItems(lineItems, [packScan({ quantity: 4 })])
    expect(missing).toEqual([])
  })

  it('reports the shortfall when under-packed', () => {
    const missing = computeMissingPackItems(lineItems, [packScan({ quantity: 3 })])
    expect(missing).toEqual([{ line_item_id: 'li-1', name: 'Beans', expected_qty: 4, packed_qty: 3 }])
  })

  it('reports the full expected quantity as missing when nothing was scanned', () => {
    const missing = computeMissingPackItems(lineItems, [])
    expect(missing).toEqual([{ line_item_id: 'li-1', name: 'Beans', expected_qty: 4, packed_qty: 0 }])
  })
})
