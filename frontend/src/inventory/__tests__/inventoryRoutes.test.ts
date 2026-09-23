import { describe, it, expect } from 'vitest'
import { parseInventoryPath, inventoryPaths } from '../inventoryRoutes'

describe('parseInventoryPath', () => {
  it.each([
    ['/inventory', { kind: 'list' }],
    ['/inventory/expiry', { kind: 'expiry' }],
    ['/inventory/products/p1/history', { kind: 'productHistory', productId: 'p1' }],
    ['/inventory/products/p1/skus/s1/history', { kind: 'skuHistory', productId: 'p1', skuId: 's1' }],
    ['/inventory/products/p1/skus/s1/batches/b1/history', { kind: 'batchHistory', productId: 'p1', skuId: 's1', batchId: 'b1' }],
    ['/inventory/products/p1', { kind: 'list' }],
    ['/inventory/nope', { kind: 'list' }],
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
