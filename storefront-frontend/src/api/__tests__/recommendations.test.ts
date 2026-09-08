import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getProductRecommendations } from '../recommendations'
import type { APIResponse, StorefrontProductSummary } from '../../types'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('recommendations api', () => {
  it('getProductRecommendations fetches the product recommendations endpoint', async () => {
    const items: StorefrontProductSummary[] = [
      { id: 'p2', slug: 'p2', name: 'Gadget', category_id: null, tags: [], image_url: null, price_minor_units: 750, currency_code: 'EUR' },
    ]
    mockFetch(items)

    const result = await getProductRecommendations('p1')

    expect(result).toEqual(items)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/products/p1/recommendations')
  })
})
