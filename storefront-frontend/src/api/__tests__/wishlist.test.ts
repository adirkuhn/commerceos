import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createWishlist, getWishlist, toggleWishlistItem } from '../wishlist'
import type { APIResponse, Wishlist } from '../../types'

function mockFetch<T>(data: T, ok = true, status = 200) {
  const body: APIResponse<T> = { data: ok ? data : null, error: ok ? null : { code: 'ERR', message: 'failed' } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok, status, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

const sampleWishlist: Wishlist = { token: 'wishlist-1', product_ids: [] }

describe('wishlist api', () => {
  it('createWishlist POSTs to /wishlists', async () => {
    mockFetch(sampleWishlist)
    const result = await createWishlist()
    expect(result).toEqual(sampleWishlist)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/storefront/v1/wishlists')
    expect(options?.method).toBe('POST')
  })

  it('getWishlist fetches by token', async () => {
    mockFetch(sampleWishlist)
    await getWishlist('wishlist-1')
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/wishlists/wishlist-1')
  })

  it('toggleWishlistItem PUTs the product id', async () => {
    mockFetch({ token: 'wishlist-1', product_ids: ['p1'] })
    await toggleWishlistItem('wishlist-1', 'p1')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/wishlists/wishlist-1/items')
    expect(options?.method).toBe('PUT')
    expect(JSON.parse(options?.body as string)).toEqual({ product_id: 'p1' })
  })

  it('throws with the server error message on failure', async () => {
    mockFetch(null, false, 404)
    await expect(getWishlist('missing-token')).rejects.toThrow('failed')
  })
})
