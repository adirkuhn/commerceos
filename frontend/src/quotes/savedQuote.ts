import { encodeQuoteLines } from './quoteLines'
import { applyDiscount, listPrice } from '../pricelist/pricing'
import type { PriceListRow } from '../pricelist/types'
import type { QuoteInput, QuoteLine, SavedQuote } from './types'

// Moving between a saved quote (API) and the quote editor (URL).

/** The editor's URL params for a saved quote: its stored prices and discount. */
export function savedQuoteParams(q: SavedQuote): Record<string, string> {
  const params: Record<string, string> = {}
  if (q.discount_basis_points) params.discount = String(q.discount_basis_points / 100)
  const lines = encodeQuoteLines(q.lines.map(l => ({
    productId: l.product_id, skuId: l.sku_id, quantity: l.quantity, unitPrice: l.unit_price_minor_units,
  })))
  if (lines) params.lines = lines
  return params
}

/** Where a saved quote opens in the editor. */
export function savedQuotePath(q: SavedQuote): string {
  const qs = new URLSearchParams(savedQuoteParams(q)).toString()
  return `/quotes/${encodeURIComponent(q.id)}${qs ? `?${qs}` : ''}`
}

/**
 * What saving the editor sends: every line at the price it shows -- typed,
 * else the discounted (sale) price. A line whose product is no longer
 * listed is kept only if its price is known.
 */
export function buildQuoteInput(
  name: string, customerId: string | null, discountPercent: number, lines: QuoteLine[],
  rowBySku: ReadonlyMap<string, PriceListRow>, fallbackCurrency?: string,
): QuoteInput {
  const out = lines.flatMap(l => {
    const row = rowBySku.get(l.skuId)
    const price = l.unitPrice ?? (row ? applyDiscount(listPrice(row), discountPercent) : undefined)
    return price === undefined ? [] : [{ product_id: l.productId, sku_id: l.skuId, quantity: l.quantity, unit_price_minor_units: price }]
  })
  const currency = lines.map(l => rowBySku.get(l.skuId)?.currency_code).find(Boolean) ?? (out.length ? fallbackCurrency : undefined)
  return {
    name: name.trim(), customer_id: customerId, discount_basis_points: Math.round(discountPercent * 100),
    currency_code: currency || undefined, lines: out,
  }
}

export function savedQuoteInput(q: SavedQuote): QuoteInput {
  return {
    name: q.name, customer_id: q.customer?.id ?? null, discount_basis_points: q.discount_basis_points, currency_code: q.currency_code || undefined,
    lines: q.lines.map(({ line_total_minor_units: _total, ...l }) => l),
  }
}

/** Unsaved changes check; currency follows the lines, so it's left out. */
export function isSameQuote(a: QuoteInput, b: QuoteInput): boolean {
  const key = (q: QuoteInput) => JSON.stringify([q.name, q.customer_id ?? null, q.discount_basis_points,
    q.lines.map(l => [l.product_id, l.sku_id, l.quantity, l.unit_price_minor_units])])
  return key(a) === key(b)
}
