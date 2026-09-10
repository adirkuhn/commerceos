import { describe, expect, it, beforeEach } from 'vitest'
import { useWishlistStore } from '../wishlistStore'

beforeEach(() => useWishlistStore.setState({ productIds: [] }))

describe('useWishlistStore', () => {
  it('adds and removes a product', () => {
    useWishlistStore.getState().add('p1')
    expect(useWishlistStore.getState().isWishlisted('p1')).toBe(true)
    useWishlistStore.getState().remove('p1')
    expect(useWishlistStore.getState().isWishlisted('p1')).toBe(false)
  })

  it('does not add the same product twice', () => {
    useWishlistStore.getState().add('p1')
    useWishlistStore.getState().add('p1')
    expect(useWishlistStore.getState().productIds).toHaveLength(1)
  })

  it('toggle flips membership', () => {
    useWishlistStore.getState().toggle('p1')
    expect(useWishlistStore.getState().isWishlisted('p1')).toBe(true)
    useWishlistStore.getState().toggle('p1')
    expect(useWishlistStore.getState().isWishlisted('p1')).toBe(false)
  })
})
