import { describe, expect, it } from 'vitest'
import { slotKey, stockSlots } from '../stockSlots'
import type { Batch } from '../../types'

const batch = (over: Partial<Batch>): Batch => ({
  id: 'b', tenant_id: 't', sku_id: 's', warehouse_id: 'w1', lot_number: '', expiry_date: null,
  received_at: '', created_at: '', updated_at: '', quantity: 0, is_general_ledger: false, ...over,
})
const warehouses = [{ id: 'w1', name: 'Main' }, { id: 'w2', name: 'Shop' }]

describe('stockSlots', () => {
  it('gives every warehouse a no-lot slot, even one without stock yet', () => {
    const slots = stockSlots(warehouses, [batch({ id: 'gl', warehouse_id: 'w1', is_general_ledger: true, quantity: 5 })])
    expect(slots.map(s => [s.warehouseName, s.batchId, s.quantity])).toEqual([['Main', '', 5], ['Shop', '', 0]])
  })

  it('lists lots after the no-lot slot, soonest expiry first, undated last', () => {
    const slots = stockSlots([warehouses[0]], [
      batch({ id: 'late', lot_number: 'L2', expiry_date: '2027-05-01', quantity: 1 }),
      batch({ id: 'none', lot_number: 'L3', quantity: 1 }),
      batch({ id: 'soon', lot_number: 'L1', expiry_date: '2027-01-01', quantity: 1 }),
    ])
    expect(slots.map(s => s.batchId)).toEqual(['', 'soon', 'late', 'none'])
    expect(slots[1]).toMatchObject({ lot: 'L1', expiryDate: '2027-01-01', isLedger: false })
  })

  it('marks lots with nothing left as empty', () => {
    const slots = stockSlots([warehouses[0]], [batch({ id: 'x', lot_number: 'L', quantity: 0 })])
    expect(slots.find(s => s.batchId === 'x')?.empty).toBe(true)
    expect(slots.find(s => s.isLedger)?.empty).toBe(false)
  })

  it('keeps a lot whose warehouse is not in the list, named by its id', () => {
    const slots = stockSlots([], [batch({ id: 'x', warehouse_id: 'gone', quantity: 2 })])
    expect(slots).toEqual([expect.objectContaining({ warehouseId: 'gone', warehouseName: 'gone', batchId: 'x' })])
  })
})

describe('slotKey', () => {
  it('tells a lot from the no-lot slot of the same warehouse', () => {
    expect(slotKey('w1', '')).not.toBe(slotKey('w1', 'b1'))
  })
})
