import { describe, it, expect } from 'vitest'
import { buildClearancePromotion, clearanceLine, previewSalePrice, urgencyOf, nextWindowOf } from '../clearance'
import type { CloseToExpiryBatch } from '../types'

const batch: CloseToExpiryBatch = {
  batch_id: 'b-1', lot_number: 'L1', expiry_date: '2026-10-30', days_left: 18, quantity: 7,
  warehouse_id: 'wh-1', warehouse_name: 'Amsterdam', sku_id: 'sku-1', sku_code: 'PQ-1',
  product_id: 'p-1', product_name: 'Pão de queijo', image_url: '', price_minor_units: 650, currency_code: 'EUR',
}

describe('clearanceLine', () => {
  it('turns a new price into a fixed_price line', () => {
    expect(clearanceLine('price', '4,99')).toEqual({ discount_type: 'fixed_price', discount_value: 499 })
  })
  it('turns a percentage into basis points', () => {
    expect(clearanceLine('percent', '30')).toEqual({ discount_type: 'percentage', discount_value: 3000 })
    expect(clearanceLine('percent', '12.5')).toEqual({ discount_type: 'percentage', discount_value: 1250 })
  })
  it('turns an amount off into a fixed_amount line', () => {
    expect(clearanceLine('amount', '1.50')).toEqual({ discount_type: 'fixed_amount', discount_value: 150 })
  })
  it('rejects empty, zero and out-of-range values', () => {
    expect(clearanceLine('price', '')).toBeNull()
    expect(clearanceLine('price', '0')).toBeNull()
    expect(clearanceLine('percent', '100.5')).toBeNull()
    expect(clearanceLine('amount', 'abc')).toBeNull()
  })
})

describe('previewSalePrice', () => {
  it('mirrors the backend: never above the price, never below one cent', () => {
    expect(previewSalePrice(650, { discount_type: 'fixed_price', discount_value: 499 })).toBe(499)
    expect(previewSalePrice(650, { discount_type: 'fixed_price', discount_value: 900 })).toBe(650)
    expect(previewSalePrice(650, { discount_type: 'percentage', discount_value: 3000 })).toBe(455)
    expect(previewSalePrice(650, { discount_type: 'fixed_amount', discount_value: 1000 })).toBe(1)
  })
})

describe('buildClearancePromotion', () => {
  it('builds an active promotion with one clearance line for the batch', () => {
    const now = new Date('2026-10-12T08:00:00Z')
    expect(buildClearancePromotion({
      batch, line: { discount_type: 'fixed_price', discount_value: 499 },
      name: '  Clearance pão de queijo ', endsAt: new Date('2026-10-30T22:00:00Z'), now,
    })).toEqual({
      name: 'Clearance pão de queijo', type: 'fixed_price', status: 'active',
      starts_at: '2026-10-12T08:00:00.000Z', ends_at: '2026-10-30T22:00:00.000Z',
      lines: [{ sku_id: 'sku-1', batch_id: 'b-1', discount_type: 'fixed_price', discount_value: 499 }],
    })
  })
  it('leaves the end open when none is given', () => {
    const p = buildClearancePromotion({ batch, line: { discount_type: 'percentage', discount_value: 1000 }, name: 'x', endsAt: null, now: new Date() })
    expect(p.ends_at).toBeNull()
  })
})

describe('urgencyOf', () => {
  it('grades by days left', () => {
    expect(urgencyOf(0)).toBe('critical')
    expect(urgencyOf(7)).toBe('critical')
    expect(urgencyOf(8)).toBe('high')
    expect(urgencyOf(15)).toBe('high')
    expect(urgencyOf(16)).toBe('medium')
    expect(urgencyOf(30)).toBe('medium')
    expect(urgencyOf(31)).toBe('low')
  })
})

describe('nextWindowOf', () => {
  const windows = [90, 30, 15, 7].map(d => ({ window_days: d }))
  it('is the next smaller window', () => {
    expect(nextWindowOf(windows, 90)).toBe(30)
    expect(nextWindowOf(windows, 15)).toBe(7)
  })
  it('is null for the smallest window', () => {
    expect(nextWindowOf(windows, 7)).toBeNull()
  })
})
