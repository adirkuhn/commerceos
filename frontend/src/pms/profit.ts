// Profit at a selling price against a cost -- shared by the product page's
// margin panel and the put-on-sale dialog so their numbers never drift.
// Same formula as the backend's GetSKUMarginInsightUseCase: margin is
// profit over price, markup profit over cost, in basis points.

export interface Profit {
  price: number
  profit: number
  marginBasisPoints: number
  markupBasisPoints: number
}

export function profitAt(priceMinorUnits: number, costMinorUnits: number): Profit {
  const profit = priceMinorUnits - costMinorUnits
  return {
    price: priceMinorUnits,
    profit,
    marginBasisPoints: priceMinorUnits > 0 ? Math.round((profit * 10000) / priceMinorUnits) : 0,
    markupBasisPoints: costMinorUnits > 0 ? Math.round((profit * 10000) / costMinorUnits) : 0,
  }
}

// profitExclIncl is the "excl. VAT / incl. VAT" pair for one stored price:
// the stored price is VAT-inclusive or -exclusive per the tenant's
// prices_include_tax, and the other side is derived with the rate. Cost
// never carries VAT (what the supplier was paid), so only price and profit
// shift. Null when the VAT rate is unknown.
export function profitExclIncl(
  storedPriceMinorUnits: number, costMinorUnits: number, taxRateBasisPoints: number | null, pricesIncludeTax: boolean,
): { exclusive: Profit; inclusive: Profit } | null {
  if (taxRateBasisPoints == null) return null
  const rate = taxRateBasisPoints / 10000
  const exclusivePrice = pricesIncludeTax ? Math.round(storedPriceMinorUnits / (1 + rate)) : storedPriceMinorUnits
  const inclusivePrice = pricesIncludeTax ? storedPriceMinorUnits : Math.round(storedPriceMinorUnits * (1 + rate))
  return { exclusive: profitAt(exclusivePrice, costMinorUnits), inclusive: profitAt(inclusivePrice, costMinorUnits) }
}
