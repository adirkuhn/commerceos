import { subPathSegments } from '../routes'

// URLs for the Inventory page's full-screen views, so a refresh or Back
// keeps them open. History views are addressed by product (plus SKU and
// batch) ids and reload what they show from those ids:
//   /inventory                                             stock list (?q=&page=&stock=)
//   /inventory/expiry                                      expiry & discards review (?tab=)
//   /inventory/products/<p>/history                        product movement history
//   /inventory/products/<p>/skus/<s>/history               SKU movement history
//   /inventory/products/<p>/skus/<s>/batches/<b>/history   batch movement history
//   /inventory/products/<p>/skus/<s>/<action>              SKU action modal over the list,
//        scoped to one batch row via ?batch=<id> or ?ledger=<warehouseId>
// The list additionally keeps expanded product rows in ?open= and SKUs
// showing zero-stock batches in ?allBatches=.
export const INVENTORY_PATH = '/inventory'

// Modals a SKU panel can open. 'choose' is the batch-row chooser (needs a
// batch scope); 'edit-batch' needs ?batch=.
export const SKU_ACTIONS = [
  'adjust', 'recount', 'transfer', 'split-case', 'portion', 'split-packages', 'damaged-lost', 'new-batch', 'price', 'choose', 'edit-batch',
] as const
export type SkuAction = typeof SKU_ACTIONS[number]

export type InventoryView =
  | { kind: 'list' }
  | { kind: 'expiry' }
  | { kind: 'productHistory'; productId: string }
  | { kind: 'skuHistory'; productId: string; skuId: string }
  | { kind: 'batchHistory'; productId: string; skuId: string; batchId: string }
  | { kind: 'skuAction'; productId: string; skuId: string; action: SkuAction }

// The stock list is shown for both the plain list and a SKU action (whose
// modal opens over the list).
export function isListView(view: InventoryView): view is Extract<InventoryView, { kind: 'list' | 'skuAction' }> {
  return view.kind === 'list' || view.kind === 'skuAction'
}

export function parseInventoryPath(pathname: string): InventoryView {
  const seg = subPathSegments(pathname, INVENTORY_PATH)
  if (seg.length === 1 && seg[0] === 'expiry') return { kind: 'expiry' }
  if (seg.length === 5 && seg[0] === 'products' && seg[2] === 'skus' && (SKU_ACTIONS as readonly string[]).includes(seg[4])) {
    return { kind: 'skuAction', productId: seg[1], skuId: seg[3], action: seg[4] as SkuAction }
  }
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
  skuAction: (productId: string, skuId: string, action: SkuAction) => `${productBase(productId)}/skus/${enc(skuId)}/${action}`,
}
