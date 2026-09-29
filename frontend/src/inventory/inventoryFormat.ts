import i18next from 'i18next'
import type { SKU } from '../pms/types'
import type { MovementType } from './types'

// No per-SKU reorder threshold exists in the backend yet -- this is a
// client-side heuristic ("low" = at or below this many units, but not
// zero) until a real configurable threshold field is added.
export const LOW_STOCK_THRESHOLD = 10

export function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

export function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
}

export function parseOptionalPriceToMinorUnits(price: string): number | null {
  if (price.trim() === '') return null
  return Math.round(parseFloat(price) * 100)
}

// formatSKUQuantity shows a kilo SKU's stock (stored in grams) as kg for
// readability -- every other SKU's stock is a plain unit count.
export function formatSKUQuantity(sku: SKU, quantity: number): string {
  if (sku.unit_type === 'kilo') return `${(quantity / 1000).toFixed(2)}kg`
  return String(quantity)
}

export function formatMinorUnitsAsInput(minorUnits: number): string {
  return (minorUnits / 100).toFixed(2)
}

export function isExpired(iso?: string | null): boolean {
  if (!iso) return false
  return new Date(iso).getTime() < Date.now()
}

export function isExpiringSoon(iso?: string | null, withinDays = 30): boolean {
  if (!iso) return false
  const ms = new Date(iso).getTime() - Date.now()
  return ms >= 0 && ms <= withinDays * 24 * 60 * 60 * 1000
}

// ── SKU movement history (full-page, entered via the "History" button —
// deliberately not an inline table in the expanded row, to keep that
// compact) ────────────────────────────────────────────────────────────────

export function movementTypeLabel(type: MovementType): string {
  switch (type) {
    case 'receipt': return i18next.t('inventory:movementType.receipt')
    case 'sale': return i18next.t('inventory:movementType.sale')
    case 'adjustment': return i18next.t('inventory:movementType.adjustment')
    case 'transfer_out': return i18next.t('inventory:movementType.transfer_out')
    case 'transfer_in': return i18next.t('inventory:movementType.transfer_in')
    case 'damaged': return i18next.t('inventory:movementType.damaged')
    case 'lost': return i18next.t('inventory:movementType.lost')
    case 'split_out': return i18next.t('inventory:movementType.split_out')
    case 'split_in': return i18next.t('inventory:movementType.split_in')
    default: return type
  }
}
