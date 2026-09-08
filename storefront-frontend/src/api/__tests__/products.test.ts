import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listProducts, getProduct } from '../products'
import type { APIResponse, StorefrontProduct, StorefrontProductListResult } from '../../types'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('products api', () => {
  it('listProducts fetches the enrolled catalog', async () => {
    const result: StorefrontProductListResult = { items: [], total: 0 }
    mockFetch(result)
    expect(await listProducts()).toEqual(result)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/products')
  })

  it('listProducts sends query params for filters/paging', async () => {
    mockFetch<StorefrontProductListResult>({ items: [], total: 0 })
    await listProducts({ categoryId: 'cat-1', search: 'shoe', page: 2, perPage: 10 })
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('category_id=cat-1')
    expect(url).toContain('search=shoe')
    expect(url).toContain('page=2')
    expect(url).toContain('per_page=10')
  })

  it('getProduct fetches one product by id', async () => {
    const product: StorefrontProduct = {
      id: 'p1', slug: 'p1', name: 'Widget', description: '', category_id: null, tags: [], images: [], variants: [],
    }
    mockFetch(product)
    expect(await getProduct('p1')).toEqual(product)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/products/p1')
  })
})
