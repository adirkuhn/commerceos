import * as pmsApi from '../pms/api'
import * as inventoryApi from './api'
import { useResource, type ResourceState } from '../routing/useResource'
import type { InventoryView } from './inventoryRoutes'
import type { SKU } from '../pms/types'
import type { Batch } from './types'

export interface InventoryHistoryTarget {
  productName: string
  skus: SKU[]
  sku: SKU | null
  batch: Batch | null
}

type HistoryView = Extract<InventoryView, { kind: 'productHistory' | 'skuHistory' | 'batchHistory' }>

// Which part of a history URL pointed at nothing -- the page shows
// "<subject> not found" rather than a generic load error.
export type HistorySubject = 'Product' | 'SKU' | 'Batch'

export class HistoryTargetNotFoundError extends Error {
  constructor(readonly subject: HistorySubject) {
    super(`${subject} not found`)
  }
}

// Reloads everything a movement-history page needs from the ids in its URL
// (see inventoryRoutes) -- the product (name + every SKU), and the SKU /
// batch the URL names -- so the page survives a refresh instead of relying
// on objects handed over from a row click.
export function useInventoryHistoryTarget(token: string, view: HistoryView): ResourceState<InventoryHistoryTarget> {
  const skuId = view.kind === 'productHistory' ? null : view.skuId
  const batchId = view.kind === 'batchHistory' ? view.batchId : null

  return useResource(async () => {
    const detail = await pmsApi.getProduct(token, view.productId)
    const skus = detail.variants.flatMap(v => v.skus ?? [])
    const sku = skuId ? skus.find(s => s.id === skuId) ?? null : null
    if (skuId && !sku) throw new HistoryTargetNotFoundError('SKU')
    let batch: Batch | null = null
    if (batchId && sku) {
      batch = (await inventoryApi.listBatches(token, sku.id)).find(b => b.id === batchId) ?? null
      if (!batch) throw new HistoryTargetNotFoundError('Batch')
    }
    return { productName: detail.product.name, skus, sku, batch }
  }, [token, view.productId, skuId, batchId])
}
