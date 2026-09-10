import type { PriceList, QuantityDiscountTier } from '@/types'

// applyPriceList is the one place a company's price-list discount turns
// into an actual minor-unit price -- every B2B price display (PDP today)
// calls this instead of recomputing the percentage itself.
export function applyPriceList(price: number, priceList: PriceList | undefined): number {
  if (!priceList) return price
  return Math.round(price * (1 - priceList.discountPercent / 100))
}

// bestQuantityBreak picks the highest-minQuantity tier a given quantity
// actually qualifies for (tiers aren't assumed to be pre-sorted).
export function bestQuantityBreak(quantity: number, tiers: QuantityDiscountTier[] | undefined): QuantityDiscountTier | undefined {
  if (!tiers || tiers.length === 0) return undefined
  return tiers
    .filter(t => quantity >= t.minQuantity)
    .sort((a, b) => b.minQuantity - a.minQuantity)[0]
}

// priceForQuantity layers a quantity-break discount (if the qty qualifies)
// on top of any price-list price -- used by quick order and the PDP
// quantity-breaks table so both agree on the same math.
export function priceForQuantity(basePrice: number, quantity: number, tiers: QuantityDiscountTier[] | undefined): number {
  const tier = bestQuantityBreak(quantity, tiers)
  if (!tier) return basePrice
  return Math.round(basePrice * (1 - tier.discountPercent / 100))
}
