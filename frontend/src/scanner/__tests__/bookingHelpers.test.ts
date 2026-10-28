import { describe, it, expect } from 'vitest'
import { defaultTakeSource, PENDING_PREFIX } from '../bookingHelpers'
import type { BookingScan } from '../types'

function scan(over: Partial<BookingScan>): BookingScan {
  return {
    id: 's', tenant_id: 't', booking_id: 'b', sku_id: 'sku-1', action_type: 'receive',
    quantity: 1, write_off_quantity: 0, scanned_at: '2026-10-28T09:00:00Z', scanned_by: 'u', ...over,
  } as BookingScan
}

describe('defaultTakeSource', () => {
  it('is the pending lot this booking received the product into', () => {
    expect(defaultTakeSource([scan({ lot_number: '31/10/2027', quantity: 47 })])).toBe(`${PENDING_PREFIX}31/10/2027`)
  })

  it('is the existing batch a receive went into', () => {
    expect(defaultTakeSource([scan({ batch_id: 'batch-9', lot_number: 'L1' })])).toBe('batch-9')
  })

  it('uses the latest receive when there are several', () => {
    expect(defaultTakeSource([
      scan({ lot_number: 'OLD', scanned_at: '2026-10-28T09:00:00Z' }),
      scan({ lot_number: 'NEW', scanned_at: '2026-10-28T09:30:00Z' }),
      scan({ lot_number: 'NEW', action_type: 'damaged', write_off_quantity: 1, quantity: 0, scanned_at: '2026-10-28T09:40:00Z' }),
    ])).toBe(`${PENDING_PREFIX}NEW`)
  })

  it('is general ledger when nothing was received, or only into an unnamed new batch', () => {
    expect(defaultTakeSource([])).toBe('')
    expect(defaultTakeSource([scan({})])).toBe('')
  })
})
