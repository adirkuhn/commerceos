import { profitExclIncl } from '../pms/profit'
import type { PriceListRow } from './types'

// Discount, profit and totals for the price list and the B2B quote. Money
// is integer minor units throughout; profit is always VAT-exclusive (cost
// never carries VAT), via the same helper as the product page's margin panel.

/** The ?discount= percentage, clamped to 0..100; anything else reads as 0. */
export function parseDiscount(raw: string | null): number {
  const n = raw === null ? NaN : Number(raw)
  if (!Number.isFinite(n)) return 0
  return Math.min(100, Math.max(0, n))
}

export function applyDiscount(priceMinor: number, discountPercent: number): number {
  const basisPoints = Math.round(discountPercent * 100)
  return Math.round((priceMinor * (10000 - basisPoints)) / 10000)
}

/** What the shop sells it for today: the sale price while one runs. */
export function listPrice(row: PriceListRow): number {
  return row.sale_price_minor_units ?? row.price_minor_units
}

export interface UnitEconomics {
  profit: number | null
  markupBasisPoints: number | null
  // Markup under the row's target -- or below cost when no target is set.
  belowTarget: boolean
}

export function unitEconomics(row: PriceListRow, unitPriceMinor: number, pricesIncludeTax: boolean): UnitEconomics {
  const cost = row.cost_price_minor_units
  if (cost == null) return { profit: null, markupBasisPoints: null, belowTarget: false }
  const { exclusive } = profitExclIncl(unitPriceMinor, cost, row.tax_rate_basis_points, pricesIncludeTax)!
  const belowTarget = cost > 0 ? exclusive.markupBasisPoints < row.target_markup_basis_points : exclusive.profit < 0
  return { profit: exclusive.profit, markupBasisPoints: cost > 0 ? exclusive.markupBasisPoints : null, belowTarget }
}

export interface PricedLine {
  row: PriceListRow
  quantity: number
  unitPrice: number
}

export interface QuoteTotals {
  total: number
  profit: number
  // False when some line has no known cost, so profit leaves it out.
  profitComplete: boolean
}

export function quoteTotals(lines: PricedLine[], pricesIncludeTax: boolean): QuoteTotals {
  return lines.reduce<QuoteTotals>((acc, line) => {
    const { profit } = unitEconomics(line.row, line.unitPrice, pricesIncludeTax)
    return {
      total: acc.total + line.unitPrice * line.quantity,
      profit: acc.profit + (profit ?? 0) * line.quantity,
      profitComplete: acc.profitComplete && profit !== null,
    }
  }, { total: 0, profit: 0, profitComplete: true })
}
