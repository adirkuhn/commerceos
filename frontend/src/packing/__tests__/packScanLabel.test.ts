import { describe, it, expect } from 'vitest'
import { packScanLabel } from '../packScanLabel'
import type { PackScan } from '../../scanner/types'

const base: PackScan = { id: 's', session_id: 'p', line_item_id: 'li', sku_id: 'sku', action_type: 'pack', quantity: 2, scanned_at: '', scanned_by: '' }

describe('packScanLabel', () => {
  it('labels each action', () => {
    expect(packScanLabel(base)).toBe('Packed · qty 2')
    expect(packScanLabel({ ...base, action_type: 'substitute' })).toBe('Substituted · qty 2')
    expect(packScanLabel({ ...base, action_type: 'refund', quantity: 1 })).toBe('Refunded · qty 1')
  })

  it('names the extra item', () => {
    expect(packScanLabel({ ...base, action_type: 'new_item', line_item_id: null, reason: 'Jam jar', quantity: 1 })).toBe('New item · Jam jar · qty 1')
    expect(packScanLabel({ ...base, action_type: 'freebie', line_item_id: null, quantity: 1 })).toBe('Freebie · qty 1')
  })
})
