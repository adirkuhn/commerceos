import { describe, it, expect } from 'vitest'
import { packProgress } from '../packProgress'
import type { PackingSessionDetail, PackScan } from '../../scanner/types'

const line = (id: string, quantity: number) => ({ id, sku_id: `sku-${id}`, name: id, sku_code: id, barcode: '', quantity, price_minor_units: 0, total_minor_units: 0 })
const pack = (lineItemId: string, quantity: number): PackScan => ({ id: `s-${lineItemId}-${quantity}`, session_id: 'p', line_item_id: lineItemId, sku_id: 'x', action_type: 'pack', quantity, scanned_at: '', scanned_by: '' })
function detail(lines: ReturnType<typeof line>[], scans: PackScan[]): PackingSessionDetail {
  return { session: { id: 'p', sales_order_id: 'so', status: 'in_progress', started_at: '', completed_by: '' }, line_items: lines, scans }
}

describe('packProgress', () => {
  it('is empty with no session', () => {
    expect(packProgress(null)).toMatchObject({ totalItems: 0, packedItems: 0, percent: 0, allPacked: false })
  })

  it('counts each line at most its ordered quantity', () => {
    const p = packProgress(detail([line('a', 2), line('b', 2)], [pack('a', 3)]))
    expect(p.packedItems).toBe(2)
    expect(p.totalItems).toBe(4)
    expect(p.percent).toBe(50)
    expect(p.allPacked).toBe(false)
    expect(p.packedByLineItem.get('a')).toBe(3)
  })

  it('is all packed once every line is covered', () => {
    expect(packProgress(detail([line('a', 1)], [pack('a', 1)])).allPacked).toBe(true)
  })

  it('an order with no lines is never "all packed"', () => {
    expect(packProgress(detail([], [])).allPacked).toBe(false)
  })
})
