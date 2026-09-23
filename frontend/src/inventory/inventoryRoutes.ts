import { subPathSegments } from '../routes'

// URLs for the Inventory page's full-screen views, so a refresh or Back
// keeps them open. History views are addressed by product (plus SKU and
// batch) ids and reload what they show from those ids:
//   /inventory                                             stock list (?q=&page=&stock=)
//   /inventory/expiry                                      expiry & discards review (?tab=)
//   /inventory/products/<p>/history                        product movement history
//   /inventory/products/<p>/skus/<s>/history               SKU movement history
//   /inventory/products/<p>/skus/<s>/batches/<b>/history   batch movement history
export const INVENTORY_PATH = '/inventory'

export type InventoryView =
  | { kind: 'list' }
  | { kind: 'expiry' }
  | { kind: 'productHistory'; productId: string }
  | { kind: 'skuHistory'; productId: string; skuId: string }
  | { kind: 'batchHistory'; productId: string; skuId: string; batchId: string }

export function parseInventoryPath(pathname: string): InventoryView {
  const seg = subPathSegments(pathname, INVENTORY_PATH)
  if (seg.length === 1 && seg[0] === 'expiry') return { kind: 'expiry' }
  if (seg[0] !== 'products' || !seg[1] || seg[seg.length - 1] !== 'history') return { kind: 'list' }
  const productId = seg[1]
  if (seg.length === 3) return { kind: 'productHistory', productId }
  if (seg.length === 5 && seg[2] === 'skus') return { kind: 'skuHistory', productId, skuId: seg[3] }
  if (seg.length === 7 && seg[2] === 'skus' && seg[4] === 'batches') {
    return { kind: 'batchHistory', productId, skuId: seg[3], batchId: seg[5] }
  }
  return { kind: 'list' }
}

const enc = encodeURIComponent
const productBase = (productId: string) => `${INVENTORY_PATH}/products/${enc(productId)}`

export const inventoryPaths = {
  list: INVENTORY_PATH,
  expiry: `${INVENTORY_PATH}/expiry`,
  productHistory: (productId: string) => `${productBase(productId)}/history`,
  skuHistory: (productId: string, skuId: string) => `${productBase(productId)}/skus/${enc(skuId)}/history`,
  batchHistory: (productId: string, skuId: string, batchId: string) =>
    `${productBase(productId)}/skus/${enc(skuId)}/batches/${enc(batchId)}/history`,
}
