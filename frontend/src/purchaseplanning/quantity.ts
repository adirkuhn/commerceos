import type { RecommendationLine } from './types'

const GRAMS_PER_KG = 1000
const SOON_DAYS = 7

function kg(grams: number): string {
  return `${Math.round((grams / GRAMS_PER_KG) * 10) / 10} kg`
}

// Stock units are grams for a weight pool, whole items otherwise.
export function formatStockQuantity(units: number, isWeightPool: boolean): string {
  return isWeightPool ? kg(units) : String(units)
}

// What to put on the purchase order: "80", a box size ("172 × 6"), or kg
// for a weight pool ("42 kg", "3 × 5 kg").
export function formatPurchaseQuantity(
  line: Pick<RecommendationLine, 'purchase_quantity' | 'units_per_purchased_item' | 'is_weight_pool'>,
): string {
  const { purchase_quantity: qty, units_per_purchased_item: perItem, is_weight_pool: pool } = line
  if (pool) return perItem === GRAMS_PER_KG ? `${qty} kg` : `${qty} × ${kg(perItem)}`
  return perItem > 1 ? `${qty} × ${perItem}` : String(qty)
}

export type OrderUrgency = 'overdue' | 'today' | 'soon' | 'later' | 'none'

// Both dates are ISO calendar days (YYYY-MM-DD), compared as UTC days.
export function orderUrgency(orderBy: string | null | undefined, asOf: string): OrderUrgency {
  if (!orderBy) return 'none'
  const days = (Date.parse(orderBy) - Date.parse(asOf)) / 86_400_000
  if (days < 0) return 'overdue'
  if (days === 0) return 'today'
  return days <= SOON_DAYS ? 'soon' : 'later'
}
