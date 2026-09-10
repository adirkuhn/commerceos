import { describe, expect, it } from 'vitest'
import { computeTotals, computeCouponDiscount, FREE_SHIPPING_THRESHOLD, STANDARD_SHIPPING, TAX_RATE } from '../cartMath'
import type { CartItem } from '@/types'

function item(overrides: Partial<CartItem> = {}): CartItem {
  return { id: 'line-1', productId: 'p1', name: 'Item', sku: 'SKU-1', price: 5000, quantity: 1, ...overrides }
}

describe('computeTotals', () => {
  it('charges standard shipping under the free-shipping threshold', () => {
    const totals = computeTotals([item({ price: 2000, quantity: 1 })], [])
    expect(totals.subtotal).toBe(2000)
    expect(totals.shipping).toBe(STANDARD_SHIPPING)
    expect(totals.tax).toBe(Math.round(2000 * TAX_RATE))
    expect(totals.total).toBe(2000 + STANDARD_SHIPPING + Math.round(2000 * TAX_RATE))
  })

  it('waives shipping once the subtotal (after discount) reaches the free-shipping threshold', () => {
    const totals = computeTotals([item({ price: FREE_SHIPPING_THRESHOLD, quantity: 1 })], [])
    expect(totals.shipping).toBe(0)
    expect(totals.freeShippingRemaining).toBe(0)
  })

  it('charges no shipping for an empty cart', () => {
    const totals = computeTotals([], [])
    expect(totals.shipping).toBe(0)
    expect(totals.subtotal).toBe(0)
  })

  it('waives shipping when a free_shipping coupon is applied, regardless of subtotal', () => {
    const totals = computeTotals([item({ price: 1000 })], [{ code: 'FREESHIP', label: 'Free shipping', discountAmount: 0 }])
    expect(totals.shipping).toBe(0)
  })

  it('computes tax on the post-discount amount, not the raw subtotal', () => {
    const totals = computeTotals([item({ price: 10000 })], [{ code: 'SAVE20', label: '$20 off', discountAmount: 2000 }])
    expect(totals.tax).toBe(Math.round(8000 * TAX_RATE))
  })
})

describe('computeCouponDiscount', () => {
  it('computes a percentage discount', () => {
    const discount = computeCouponDiscount(10000, [item({ price: 10000 })], 'WELCOME10')
    expect(discount).toBe(1000)
  })

  it('returns 0 below the minimum subtotal for a fixed coupon', () => {
    const discount = computeCouponDiscount(5000, [item({ price: 5000 })], 'SAVE20')
    expect(discount).toBe(0)
  })

  it('applies a fixed discount at or above the minimum subtotal', () => {
    const discount = computeCouponDiscount(15000, [item({ price: 15000 })], 'SAVE20')
    expect(discount).toBe(2000)
  })

  it('returns 0 for an unknown coupon code', () => {
    expect(computeCouponDiscount(10000, [item()], 'NOT-REAL')).toBe(0)
  })

  it('gives the cheapest item free for a buy-x-get-y coupon once the group size is met', () => {
    const items = [item({ price: 3000, quantity: 2 }), item({ id: 'line-2', price: 1000, quantity: 1 })]
    // 3 total units, buy 2 get 1 -- one group met, cheapest unit price (1000) is free.
    const discount = computeCouponDiscount(7000, items, 'BOGOTEE')
    expect(discount).toBe(1000)
  })
})
