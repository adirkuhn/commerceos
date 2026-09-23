import { describe, it, expect } from 'vitest'
import { parseInventoryPath, inventoryPaths, isListView, SKU_ACTIONS } from '../inventoryRoutes'

describe('parseInventoryPath', () => {
  it.each([
    ['/inventory', { kind: 'list' }],
    ['/inventory/expiry', { kind: 'expiry' }],
    ['/inventory/products/p1/history', { kind: 'productHistory', productId: 'p1' }],
    ['/inventory/products/p1/skus/s1/history', { kind: 'skuHistory', productId: 'p1', skuId: 's1' }],
    ['/inventory/products/p1/skus/s1/batches/b1/history', { kind: 'batchHistory', productId: 'p1', skuId: 's1', batchId: 'b1' }],
    ['/inventory/products/p1', { kind: 'list' }],
    ['/inventory/nope', { kind: 'list' }],
    ['/inventory/products/p1/skus/s1/adjust', { kind: 'skuAction', productId: 'p1', skuId: 's1', action: 'adjust' }],
    ['/inventory/products/p1/skus/s1/edit-batch', { kind: 'skuAction', productId: 'p1', skuId: 's1', action: 'edit-batch' }],
    ['/inventory/products/p1/skus/s1/bogus', { kind: 'list' }],
  ])('%s -> %o', (path, view) => {
    expect(parseInventoryPath(path)).toEqual(view)
  })
})

describe('inventoryPaths', () => {
  it('round-trips through the parser', () => {
    expect(parseInventoryPath(inventoryPaths.productHistory('p 1'))).toEqual({ kind: 'productHistory', productId: 'p 1' })
    expect(parseInventoryPath(inventoryPaths.skuHistory('p1', 's1'))).toEqual({ kind: 'skuHistory', productId: 'p1', skuId: 's1' })
    expect(parseInventoryPath(inventoryPaths.batchHistory('p1', 's1', 'b1'))).toEqual({ kind: 'batchHistory', productId: 'p1', skuId: 's1', batchId: 'b1' })
    expect(parseInventoryPath(inventoryPaths.expiry)).toEqual({ kind: 'expiry' })
  })
})

describe('skuAction paths', () => {
  it('round-trip through the parser', () => {
    for (const action of SKU_ACTIONS) {
      expect(parseInventoryPath(inventoryPaths.skuAction('p1', 's1', action))).toEqual({ kind: 'skuAction', productId: 'p1', skuId: 's1', action })
    }
  })

  it('an action view still belongs to the list (the modal opens over it)', () => {
    expect(isListView(parseInventoryPath('/inventory/products/p1/skus/s1/adjust'))).toBe(true)
    expect(isListView(parseInventoryPath('/inventory'))).toBe(true)
    expect(isListView(parseInventoryPath('/inventory/expiry'))).toBe(false)
  })
})
