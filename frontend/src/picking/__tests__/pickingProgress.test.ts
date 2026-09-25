import { describe, it, expect } from 'vitest'
import { pickProgress } from '../pickingProgress'
import type { PickScan, PickingSessionDetail } from '../../scanner/types'

function scan(lineItemId: string, quantity: number): PickScan {
  return { id: `s-${lineItemId}-${quantity}`, session_id: 'ps-1', line_item_id: lineItemId, sku_id: 'sku', action_type: 'pick', quantity, scanned_at: '', scanned_by: 'me' }
}

function detail(scans: PickScan[]): PickingSessionDetail {
  return {
    session: { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [
      { id: 'a', sku_id: 'sku-a', name: 'A', sku_code: 'A', barcode: '', quantity: 2, price_minor_units: 0, total_minor_units: 0 },
      { id: 'b', sku_id: 'sku-b', name: 'B', sku_code: 'B', barcode: '', quantity: 2, price_minor_units: 0, total_minor_units: 0 },
    ],
    scans,
  }
}

describe('pickProgress', () => {
  it('is empty with no session', () => {
    expect(pickProgress(null)).toMatchObject({ totalItems: 0, pickedItems: 0, percent: 0, allPicked: false })
  })

  it('counts picked units against the ordered total', () => {
    const p = pickProgress(detail([scan('a', 1)]))
    expect(p).toMatchObject({ totalItems: 4, pickedItems: 1, percent: 25, allPicked: false })
    expect(p.pickedByLineItem.get('a')).toBe(1)
  })

  it('caps an over-picked line at its quantity so it cannot hide another gap', () => {
    const p = pickProgress(detail([scan('a', 4)]))
    expect(p).toMatchObject({ pickedItems: 2, allPicked: false })
    expect(p.pickedByLineItem.get('a')).toBe(4)
  })

  it('is all picked once every line is covered', () => {
    expect(pickProgress(detail([scan('a', 2), scan('b', 2)]))).toMatchObject({ percent: 100, allPicked: true })
  })
})
