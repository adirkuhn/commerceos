import { describe, expect, it } from 'vitest'
import { ADJUST_REASONS, adjustmentPreview, buildAdjustment, reasonsFor } from '../adjustment'
import { stockUnitFor } from '../../stockUnit'

const units = stockUnitFor()
const kg = stockUnitFor({ unit_type: 'kilo' })

describe('reasonsFor', () => {
  it('offers incoming reasons for add and outgoing reasons for remove', () => {
    expect(reasonsFor('add')).toEqual(['received', 'returned', 'found', 'correction'])
    expect(reasonsFor('remove')).toEqual(['sold', 'sample', 'correction'])
  })

  it('maps every reason to a movement type the backend accepts for its sign', () => {
    expect(ADJUST_REASONS.received.type).toBe('receipt')
    expect(ADJUST_REASONS.returned.type).toBe('receipt')
    expect(ADJUST_REASONS.sold.type).toBe('sale')
    expect(ADJUST_REASONS.found.type).toBe('adjustment')
    expect(ADJUST_REASONS.sample.type).toBe('adjustment')
    expect(ADJUST_REASONS.correction.type).toBe('adjustment')
  })
})

describe('adjustmentPreview', () => {
  it('adds or removes the parsed quantity from what is on hand', () => {
    expect(adjustmentPreview({ onHand: 12, direction: 'add', quantity: '3', unit: units })).toEqual({ before: 12, after: 15, delta: 3, short: false })
    expect(adjustmentPreview({ onHand: 12, direction: 'remove', quantity: '3', unit: units })).toEqual({ before: 12, after: 9, delta: -3, short: false })
  })

  it('flags a removal larger than what is on hand', () => {
    expect(adjustmentPreview({ onHand: 2, direction: 'remove', quantity: '5', unit: units })).toMatchObject({ after: -3, short: true })
  })

  it('reads kilos for a weight pool and works in grams', () => {
    expect(adjustmentPreview({ onHand: 1500, direction: 'remove', quantity: '0,25', unit: kg })).toMatchObject({ after: 1250, delta: -250 })
  })

  it('has no preview until a positive quantity is typed', () => {
    expect(adjustmentPreview({ onHand: 4, direction: 'add', quantity: '', unit: units })).toBeNull()
    expect(adjustmentPreview({ onHand: 4, direction: 'add', quantity: '0', unit: units })).toBeNull()
    expect(adjustmentPreview({ onHand: 4, direction: 'add', quantity: '-2', unit: units })).toBeNull()
    expect(adjustmentPreview({ onHand: 4, direction: 'add', quantity: 'abc', unit: units })).toBeNull()
  })
})

describe('buildAdjustment', () => {
  const base = { warehouseId: 'w1', batchId: 'b1', unit: units, reasonLabel: 'Received' }

  it('builds a receipt with a positive delta', () => {
    expect(buildAdjustment({ ...base, direction: 'add', quantity: '4', reason: 'received', note: '' })).toEqual({
      ok: true,
      input: { warehouse_id: 'w1', batch_id: 'b1', type: 'receipt', quantity_delta: 4, reason: 'Received' },
    })
  })

  it('builds a sale with a negative delta and appends the note', () => {
    expect(buildAdjustment({ ...base, reasonLabel: 'Sold', direction: 'remove', quantity: '2', reason: 'sold', note: ' market stall ' })).toEqual({
      ok: true,
      input: { warehouse_id: 'w1', batch_id: 'b1', type: 'sale', quantity_delta: -2, reason: 'Sold — market stall' },
    })
  })

  it('sends a null batch for the general ledger', () => {
    const r = buildAdjustment({ ...base, batchId: '', direction: 'add', quantity: '1', reason: 'received', note: '' })
    expect(r.ok && r.input.batch_id).toBeNull()
  })

  it('rejects a missing or non-positive quantity', () => {
    expect(buildAdjustment({ ...base, direction: 'add', quantity: '', reason: 'received', note: '' })).toEqual({ ok: false, error: 'quantity' })
    expect(buildAdjustment({ ...base, direction: 'add', quantity: '-1', reason: 'received', note: '' })).toEqual({ ok: false, error: 'quantity' })
  })

  it('rejects a reason that does not fit the direction', () => {
    expect(buildAdjustment({ ...base, direction: 'add', quantity: '1', reason: 'sold', note: '' })).toEqual({ ok: false, error: 'reason' })
  })

  it('rejects a missing warehouse', () => {
    expect(buildAdjustment({ ...base, warehouseId: '', direction: 'add', quantity: '1', reason: 'received', note: '' })).toEqual({ ok: false, error: 'warehouse' })
  })
})
