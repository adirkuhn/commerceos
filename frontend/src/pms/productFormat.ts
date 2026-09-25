import type { ProductStatus, SKUStatus } from './types'

export const STATUS_STYLES: Record<ProductStatus, string> = {
  active: 'bg-green-500/15 text-success',
  archived: 'bg-gray-500/15 text-ink-4',
  draft: 'bg-amber-500/15 text-warning',
}

export const SKU_STATUS_STYLES: Record<SKUStatus, string> = {
  active: 'bg-green-500/15 text-success',
  inactive: 'bg-gray-500/15 text-ink-4',
}

export const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

export function formatPrice(minorUnits: number, currency: string): string {
  return `${minorUnits < 0 ? '-' : ''}${currency} ${(Math.abs(minorUnits) / 100).toFixed(2)}`
}

export function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
}

export function parseOptionalPriceToMinorUnits(price: string): number | null {
  if (price.trim() === '') return null
  return Math.round(parseFloat(price) * 100)
}

export function formatMinorUnitsAsInput(minorUnits: number): string {
  return (minorUnits / 100).toFixed(2)
}

export function parseIntOrUndefined(value: string): number | undefined {
  if (value === '') return undefined
  const n = parseInt(value, 10)
  return Number.isNaN(n) ? undefined : n
}

// SKU.ScaleCode is the 7-digit segment DecodeWeightEmbeddedBarcode reads
// out of a scanned barcode's [0:7] -- the 2-digit GS1 prefix plus the
// 5-digit code, stored and matched exactly as staff read it off the
// physical scale's own config screen. No stripping: what's typed is what's
// stored.
export function normalizeScaleCodeInput(raw: string): string {
  return raw.replace(/\D/g, '').slice(0, 7)
}
