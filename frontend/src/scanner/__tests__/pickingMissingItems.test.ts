import { describe, it, expect } from 'vitest'
import { pickedQuantityByLineItem, computeMissingItems } from '../pickingMissingItems'
import type { PickScan, PickingSessionLineItem } from '../types'

function pickScan(overrides: Partial<PickScan> = {}): PickScan {
  return {
    id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1',
    action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:00:00Z', scanned_by: 'user-1',
    ...overrides,
  }
}

describe('pickedQuantityByLineItem', () => {
  it('sums plain pick scans by their own quantity', () => {
    const result = pickedQuantityByLineItem([pickScan({ quantity: 3 })])
    expect(result.get('li-1')).toBe(3)
  })

  // Regression test for a real bug: a substitute scan's `quantity` is the
  // SUBSTITUTE sku's own quantity (e.g. 4 cheap units), not how much of the
  // original line it covers -- that's original_quantity_covered (e.g.
  // those 4 units covering only 1 original unit). This function used to
  // sum `quantity` for substitute rows too, badly overcounting "picked so
  // far" and forcing a wrong "remaining" total in both picking UIs.
  it('sums substitute scans by original_quantity_covered, not quantity', () => {
    const result = pickedQuantityByLineItem([
      pickScan({ action_type: 'substitute', quantity: 4, original_quantity_covered: 1 }),
    ])
    expect(result.get('li-1')).toBe(1)
  })

  it('combines pick and substitute scans on the same line', () => {
    const result = pickedQuantityByLineItem([
      pickScan({ id: 'scan-1', action_type: 'pick', quantity: 3 }),
      pickScan({ id: 'scan-2', action_type: 'substitute', quantity: 4, original_quantity_covered: 1 }),
    ])
    expect(result.get('li-1')).toBe(4)
  })

  it('ignores freebie/new_item scans (no line_item_id) and refund scans', () => {
    const result = pickedQuantityByLineItem([
      pickScan({ id: 'scan-1', action_type: 'freebie', line_item_id: null, quantity: 1 }),
      pickScan({ id: 'scan-2', action_type: 'refund', quantity: 1 }),
    ])
    expect(result.get('li-1')).toBeUndefined()
  })
})

describe('computeMissingItems', () => {
  const lineItems: PickingSessionLineItem[] = [
    { id: 'li-1', sku_id: 'sku-1', name: 'Beans', sku_code: 'BEANS-1', barcode: '111', quantity: 4, price_minor_units: 100, total_minor_units: 400 },
  ]

  it('reports nothing missing once pick + substitute coverage reaches the full quantity', () => {
    const missing = computeMissingItems(lineItems, [
      pickScan({ id: 'scan-1', action_type: 'pick', quantity: 3 }),
      pickScan({ id: 'scan-2', action_type: 'substitute', quantity: 1, original_quantity_covered: 1 }),
    ])
    expect(missing).toEqual([])
  })

  it('reports the correct shortfall when only substitute coverage is under-counted', () => {
    const missing = computeMissingItems(lineItems, [
      pickScan({ id: 'scan-1', action_type: 'pick', quantity: 3 }),
    ])
    expect(missing).toEqual([{ line_item_id: 'li-1', name: 'Beans', expected_qty: 4, picked_qty: 3 }])
  })
})
