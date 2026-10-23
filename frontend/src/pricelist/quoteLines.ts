import type { QuoteLine } from './types'

// A quote lives in the URL (?lines=) so refresh and Back keep it:
// product.sku.quantity[.unitPrice] per line, lines comma-separated. Ids are
// UUIDs, so neither "." nor "," can appear inside one.

const WHOLE = /^\d+$/

export function encodeQuoteLines(lines: QuoteLine[]): string | null {
  if (lines.length === 0) return null
  return lines.map(l => [l.productId, l.skuId, l.quantity, ...(l.unitPrice === undefined ? [] : [l.unitPrice])].join('.')).join(',')
}

export function decodeQuoteLines(raw: string | null): QuoteLine[] {
  if (!raw) return []
  return raw.split(',').flatMap(entry => {
    const [productId, skuId, quantity, unitPrice, ...rest] = entry.split('.')
    if (!productId || !skuId || !WHOLE.test(quantity ?? '') || rest.length > 0) return []
    if (unitPrice !== undefined && !WHOLE.test(unitPrice)) return []
    const line: QuoteLine = { productId, skuId, quantity: Number(quantity) }
    if (unitPrice !== undefined) line.unitPrice = Number(unitPrice)
    return [line]
  })
}
