import i18next from 'i18next'
import type { PriceListRow } from './types'

/** Stock of a weight pool is grams; everything else is a count. */
export function formatQuantity(row: Pick<PriceListRow, 'unit_type'>, quantity: number): string {
  return row.unit_type === 'kilo' ? i18next.t('priceList:grams', { value: quantity }) : String(quantity)
}

export function formatPercent(basisPoints: number): string {
  return `${(basisPoints / 100).toFixed(1)}%`
}

/** Why a price is flagged: under the markup target, or under cost. */
export function belowTargetText(row: Pick<PriceListRow, 'target_markup_basis_points'>, profit: number | null): string {
  return row.target_markup_basis_points > 0 && (profit ?? 0) >= 0
    ? i18next.t('priceList:belowTarget', { target: row.target_markup_basis_points / 100 })
    : i18next.t('priceList:belowCost')
}
