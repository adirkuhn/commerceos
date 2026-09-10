import { describe, expect, it } from 'vitest'
import { MockCartRepository } from '../MockCartRepository'

const repo = new MockCartRepository()

function item(overrides: Partial<Parameters<MockCartRepository['addItem']>[1]> = {}) {
  return { productId: 'p1', name: 'Trail Runner', sku: 'SKU-1', price: 5000, quantity: 1, ...overrides }
}

// Every test picks its own random cart id -- MockCartRepository's state is
// a module-level Map keyed by cartId (see its own doc comment), so a fresh
// id per test is how these stay isolated without a reset method.
describe('MockCartRepository', () => {
  it('starts empty for a new cart id', async () => {
    const cart = await repo.get(`empty-${Math.random()}`)
    expect(cart.items).toHaveLength(0)
    expect(cart.totals.total).toBe(0)
  })

  it('adds an item', async () => {
    const id = `add-${Math.random()}`
    const cart = await repo.addItem(id, item())
    expect(cart.items).toHaveLength(1)
    expect(cart.items[0].quantity).toBe(1)
  })

  it('merges quantity when adding the same product+variant again', async () => {
    const id = `merge-${Math.random()}`
    await repo.addItem(id, item({ quantity: 1 }))
    const cart = await repo.addItem(id, item({ quantity: 2 }))
    expect(cart.items).toHaveLength(1)
    expect(cart.items[0].quantity).toBe(3)
  })

  it('keeps distinct variants as separate lines', async () => {
    const id = `variant-${Math.random()}`
    await repo.addItem(id, item({ variantId: 'v1' }))
    const cart = await repo.addItem(id, item({ variantId: 'v2' }))
    expect(cart.items).toHaveLength(2)
  })

  it('updates quantity, removing the line at 0', async () => {
    const id = `qty-${Math.random()}`
    const added = await repo.addItem(id, item())
    const lineId = added.items[0].id
    const updated = await repo.updateQuantity(id, lineId, 3)
    expect(updated.items[0].quantity).toBe(3)
    const removed = await repo.updateQuantity(id, lineId, 0)
    expect(removed.items).toHaveLength(0)
  })

  it('moves an item to saved-for-later and back', async () => {
    const id = `save-${Math.random()}`
    const added = await repo.addItem(id, item())
    const lineId = added.items[0].id
    const saved = await repo.saveForLater(id, lineId)
    expect(saved.items).toHaveLength(0)
    expect(saved.savedItems).toHaveLength(1)
    const moved = await repo.moveToCart(id, saved.savedItems[0].id)
    expect(moved.items).toHaveLength(1)
    expect(moved.savedItems).toHaveLength(0)
  })

  it('applies a valid coupon and rejects an unknown one', async () => {
    const id = `coupon-${Math.random()}`
    await repo.addItem(id, item({ price: 10000 }))
    const applied = await repo.applyCoupon(id, 'welcome10')
    expect(applied.coupons).toHaveLength(1)
    expect(applied.coupons[0].discountAmount).toBe(1000)
    await expect(repo.applyCoupon(id, 'NOT-REAL')).rejects.toThrow()
  })

  it('rejects applying the same coupon twice', async () => {
    const id = `dup-coupon-${Math.random()}`
    await repo.addItem(id, item({ price: 10000 }))
    await repo.applyCoupon(id, 'WELCOME10')
    await expect(repo.applyCoupon(id, 'WELCOME10')).rejects.toThrow()
  })

  it('rejects a coupon below its minimum subtotal', async () => {
    const id = `min-sub-${Math.random()}`
    await repo.addItem(id, item({ price: 1000 }))
    await expect(repo.applyCoupon(id, 'SAVE20')).rejects.toThrow()
  })

  it('removes a coupon', async () => {
    const id = `remove-coupon-${Math.random()}`
    await repo.addItem(id, item({ price: 10000 }))
    await repo.applyCoupon(id, 'WELCOME10')
    const cart = await repo.removeCoupon(id, 'WELCOME10')
    expect(cart.coupons).toHaveLength(0)
  })

  it('clears the cart', async () => {
    const id = `clear-${Math.random()}`
    await repo.addItem(id, item())
    const cleared = await repo.clear(id)
    expect(cleared.items).toHaveLength(0)
    expect(cleared.coupons).toHaveLength(0)
  })
})
