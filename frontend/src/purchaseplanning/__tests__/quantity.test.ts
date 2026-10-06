import { describe, it, expect } from 'vitest'
import { formatStockQuantity, formatPurchaseQuantity, orderUrgency } from '../quantity'

describe('formatStockQuantity', () => {
  it('shows units as a plain count', () => {
    expect(formatStockQuantity(1031, false)).toBe('1031')
  })

  it('shows a weight pool\'s grams in kg', () => {
    expect(formatStockQuantity(12300, true)).toBe('12.3 kg')
    expect(formatStockQuantity(42000, true)).toBe('42 kg')
    expect(formatStockQuantity(-500, true)).toBe('-0.5 kg')
  })
})

describe('formatPurchaseQuantity', () => {
  it('is a plain count for single items', () => {
    expect(formatPurchaseQuantity({ purchase_quantity: 80, units_per_purchased_item: 1, is_weight_pool: false })).toBe('80')
  })

  it('shows boxes as count x box size', () => {
    expect(formatPurchaseQuantity({ purchase_quantity: 172, units_per_purchased_item: 6, is_weight_pool: false })).toBe('172 × 6')
  })

  it('shows a weight pool bought per kg in kg', () => {
    expect(formatPurchaseQuantity({ purchase_quantity: 42, units_per_purchased_item: 1000, is_weight_pool: true })).toBe('42 kg')
  })

  it('shows a weight pool bought in other packs as count x pack weight', () => {
    expect(formatPurchaseQuantity({ purchase_quantity: 3, units_per_purchased_item: 5000, is_weight_pool: true })).toBe('3 × 5 kg')
  })
})

describe('orderUrgency', () => {
  const asOf = '2026-10-06'

  it('is overdue when the order-by date is before today', () => {
    expect(orderUrgency('2026-10-05', asOf)).toBe('overdue')
  })

  it('is today on the order-by date itself', () => {
    expect(orderUrgency('2026-10-06', asOf)).toBe('today')
  })

  it('is soon within a week', () => {
    expect(orderUrgency('2026-10-13', asOf)).toBe('soon')
  })

  it('is later beyond a week', () => {
    expect(orderUrgency('2026-10-14', asOf)).toBe('later')
  })

  it('is none without a date', () => {
    expect(orderUrgency(null, asOf)).toBe('none')
    expect(orderUrgency(undefined, asOf)).toBe('none')
  })
})
