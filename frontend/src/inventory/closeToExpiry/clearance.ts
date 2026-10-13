import { toMinorUnits } from '../../collections/form'
import type { PromotionInput, PromotionLineInput, PromotionType } from '../../promotions/types'
import { profitAt, profitExclIncl, type Profit } from '../../pms/profit'
import type { CloseToExpiryBatch, CloseToExpiryWindow, UnitCostSource } from './types'

// How a clearance sale is priced: a new price, a percentage off or an
// amount off.
export type PriceMode = 'price' | 'percent' | 'amount'

export const PRICE_MODES: readonly PriceMode[] = ['price', 'percent', 'amount']

type LineDiscount = Pick<PromotionLineInput, 'discount_type' | 'discount_value'>

const TYPE_OF_MODE: Record<PriceMode, PromotionType> = {
  price: 'fixed_price', percent: 'percentage', amount: 'fixed_amount',
}

/** What the person typed, as a promotion line's discount; null when invalid. */
export function clearanceLine(mode: PriceMode, value: string): LineDiscount | null {
  if (mode === 'percent') {
    const n = Number(value.replace(',', '.'))
    const bps = Math.round(n * 100)
    if (value.trim() === '' || !Number.isFinite(n) || bps < 1 || bps > 10000) return null
    return { discount_type: TYPE_OF_MODE.percent, discount_value: bps }
  }
  const minor = toMinorUnits(value)
  if (minor === null || minor < 1) return null
  return { discount_type: TYPE_OF_MODE[mode], discount_value: minor }
}

/** The sale price a line gives a SKU -- same rule as the backend's apply step. */
export function previewSalePrice(regularMinorUnits: number, line: LineDiscount): number {
  let price = regularMinorUnits
  switch (line.discount_type) {
    case 'fixed_price': price = Math.min(line.discount_value, regularMinorUnits); break
    case 'percentage': price = regularMinorUnits - Math.floor(regularMinorUnits * line.discount_value / 10000); break
    case 'fixed_amount': price = regularMinorUnits - line.discount_value; break
  }
  return Math.max(price, 1)
}

/** An active promotion selling off one batch, starting now. */
export function buildClearancePromotion({ batch, line, name, endsAt, now }: {
  batch: CloseToExpiryBatch
  line: LineDiscount
  name: string
  endsAt: Date | null
  now: Date
}): PromotionInput {
  return {
    name: name.trim(), type: line.discount_type, status: 'active',
    starts_at: now.toISOString(), ends_at: endsAt ? endsAt.toISOString() : null,
    lines: [{ sku_id: batch.sku_id, batch_id: batch.batch_id, ...line }],
  }
}

/** The shop's VAT for this SKU: rate (null = unknown) and whether stored prices include it. */
export interface SkuVat {
  rateBasisPoints: number | null
  pricesIncludeTax: boolean
}

/** Profit at one price: as stored, plus excl./incl. VAT when the rate is known. */
export interface PriceProfit {
  stored: Profit
  exclusive: Profit | null
  inclusive: Profit | null
}

export interface ClearanceProfit {
  cost: number
  costSource: UnitCostSource
  regular: PriceProfit
  sale: PriceProfit | null
  /** Profit on everything left in the lot at the sale price -- excl. VAT when known. */
  lotProfit: number | null
  belowCost: boolean
}

function priceProfit(price: number, cost: number, vat: SkuVat): PriceProfit {
  const both = profitExclIncl(price, cost, vat.rateBasisPoints, vat.pricesIncludeTax)
  return { stored: profitAt(price, cost), exclusive: both?.exclusive ?? null, inclusive: both?.inclusive ?? null }
}

/** How much a clearance still makes, against the lot's cost (else the SKU's
 *  average). Same formula as the product page's margin panel. Null without a cost. */
export function clearanceProfit(
  batch: Pick<CloseToExpiryBatch, 'quantity' | 'price_minor_units' | 'unit_cost_minor_units' | 'unit_cost_source'>,
  salePriceMinorUnits: number | null,
  vat: SkuVat,
): ClearanceProfit | null {
  if (batch.unit_cost_minor_units == null || !batch.unit_cost_source) return null
  const cost = batch.unit_cost_minor_units
  const sale = salePriceMinorUnits == null ? null : priceProfit(salePriceMinorUnits, cost, vat)
  const realSaleProfit = sale ? (sale.exclusive ?? sale.stored).profit : null
  return {
    cost, costSource: batch.unit_cost_source,
    regular: priceProfit(batch.price_minor_units, cost, vat),
    sale,
    lotProfit: realSaleProfit == null ? null : realSaleProfit * batch.quantity,
    belowCost: realSaleProfit != null && realSaleProfit < 0,
  }
}

export type Urgency = 'critical' | 'high' | 'medium' | 'low'

/** How pressing a batch is, by days left -- drives the stamp's colour. */
export function urgencyOf(daysLeft: number): Urgency {
  if (daysLeft <= 7) return 'critical'
  if (daysLeft <= 15) return 'high'
  if (daysLeft <= 30) return 'medium'
  return 'low'
}

/** The window a batch moves into after windowDays; null for the last one. */
export function nextWindowOf(windows: Pick<CloseToExpiryWindow, 'window_days'>[], windowDays: number): number | null {
  const i = windows.findIndex(w => w.window_days === windowDays)
  return i >= 0 && i + 1 < windows.length ? windows[i + 1].window_days : null
}
