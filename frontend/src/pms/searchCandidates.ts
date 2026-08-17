import type { SKU } from './types'
import * as pmsApi from './api'

export interface SKUCandidate {
  sku: SKU
  productId: string
  productName: string
}

const DEFAULT_CANDIDATE_LIMIT = 8

// Falls back to a partial (substring) match on product name, SKU code, or
// barcode -- e.g. a scanner/camera that misread and dropped digits from a
// barcode, or a product name typed by hand. Reuses ListProducts' Search,
// which already ILIKEs sku_code/barcode as well as name/slug (see
// pms_repo.go's List), so no separate backend endpoint is needed. Every
// scan entry point (Inventory, Price Lookup, Booking) falls back to this
// the same way: exact code lookup first, this on failure.
export async function searchSKUCandidates(token: string, query: string, limit = DEFAULT_CANDIDATE_LIMIT): Promise<SKUCandidate[]> {
  const { products } = await pmsApi.listProducts(token, { search: query, perPage: limit })
  if (products.length === 0) return []
  const details = await Promise.all(products.map(p => pmsApi.getProduct(token, p.id)))
  return details.flatMap(d =>
    d.variants.flatMap(v => (v.skus ?? []).map(s => ({ sku: s, productId: d.product.id, productName: d.product.name }))))
}
