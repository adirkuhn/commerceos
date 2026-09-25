import type { TaxClass, TaxRate } from '../tax/types'

// Resolves which tax rate applies to a SKU in the tenant's default tax
// country: the SKU's own tax class if it has one, else the tenant's default
// class (see SalesOrders.tsx's lineTax for the same
// per-entity-override-with-tenant-fallback pattern).
export function resolveSkuTaxRate(
  rates: TaxRate[], classes: TaxClass[], taxClassId: string | null | undefined, country: string | null | undefined,
): TaxRate | undefined {
  const effectiveTaxClassId = taxClassId ?? classes.find(c => c.is_default)?.id ?? null
  return effectiveTaxClassId && country
    ? rates.find(r => r.tax_class_id === effectiveTaxClassId && r.country === country)
    : undefined
}

export interface TaxPreview {
  exclusive: number
  inclusive: number
  ratePercent: string
  country: string
}

// The "Excl. tax / Incl. tax" hint under a price input -- the entered price
// is taken as tax-inclusive or -exclusive per the tenant's prices_include_tax.
export function computeTaxPreview(priceMinorUnits: number | null, rate: TaxRate | undefined, pricesIncludeTax: boolean | undefined): TaxPreview | null {
  if (!rate || priceMinorUnits == null) return null
  const r = rate.rate_basis_points / 10000
  const inclusiveMinor = pricesIncludeTax ? priceMinorUnits : Math.round(priceMinorUnits * (1 + r))
  const exclusiveMinor = pricesIncludeTax ? Math.round(priceMinorUnits / (1 + r)) : priceMinorUnits
  return {
    exclusive: exclusiveMinor / 100,
    inclusive: inclusiveMinor / 100,
    ratePercent: (rate.rate_basis_points / 100).toFixed(2),
    country: rate.country,
  }
}
