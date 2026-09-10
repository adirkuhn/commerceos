import { describe, expect, it } from 'vitest'
import { applyPriceList, bestQuantityBreak, priceForQuantity } from '../b2bPricing'
import type { PriceList, QuantityDiscountTier } from '@/types'

const wholesale: PriceList = { id: 'pl-1', name: 'Wholesale', discountPercent: 20 }
const tiers: QuantityDiscountTier[] = [
  { minQuantity: 12, discountPercent: 10 },
  { minQuantity: 50, discountPercent: 20 },
  { minQuantity: 100, discountPercent: 28 },
]

describe('applyPriceList', () => {
  it('discounts the price by the price list percentage', () => {
    expect(applyPriceList(10000, wholesale)).toBe(8000)
  })

  it('returns the price unchanged when there is no price list', () => {
    expect(applyPriceList(10000, undefined)).toBe(10000)
  })
})

describe('bestQuantityBreak', () => {
  it('returns undefined below the lowest tier', () => {
    expect(bestQuantityBreak(5, tiers)).toBeUndefined()
  })

  it('picks the highest tier the quantity qualifies for', () => {
    expect(bestQuantityBreak(12, tiers)?.discountPercent).toBe(10)
    expect(bestQuantityBreak(49, tiers)?.discountPercent).toBe(10)
    expect(bestQuantityBreak(50, tiers)?.discountPercent).toBe(20)
    expect(bestQuantityBreak(150, tiers)?.discountPercent).toBe(28)
  })

  it('is not confused by unsorted tiers', () => {
    const shuffled = [tiers[2], tiers[0], tiers[1]]
    expect(bestQuantityBreak(60, shuffled)?.discountPercent).toBe(20)
  })

  it('returns undefined when there are no tiers', () => {
    expect(bestQuantityBreak(500, undefined)).toBeUndefined()
    expect(bestQuantityBreak(500, [])).toBeUndefined()
  })
})

describe('priceForQuantity', () => {
  it('applies the qualifying tier on top of the given base price', () => {
    expect(priceForQuantity(10000, 60, tiers)).toBe(8000)
  })

  it('returns the base price unchanged when quantity qualifies for no tier', () => {
    expect(priceForQuantity(10000, 1, tiers)).toBe(10000)
  })

  it('composes with a price-list-adjusted base price', () => {
    const companyPrice = applyPriceList(10000, wholesale) // 8000
    expect(priceForQuantity(companyPrice, 100, tiers)).toBe(Math.round(8000 * 0.72))
  })
})
