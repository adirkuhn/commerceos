// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { APIResponse, Category, Product, ProductDetail, Variant, SKU, Media } from '../types'

const TOKEN = 'test-token'
const BASE = 'https://api.commerceos.dev.localhost/api/v1/pms'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const sampleCategory: Category = {
  id: 'c1', tenant_id: 't1', name: 'Electronics', slug: 'electronics',
  position: 0, created_at: '2026-07-29T00:00:00Z', updated_at: '2026-07-29T00:00:00Z',
}

const sampleProduct: Product = {
  id: 'p1', tenant_id: 't1', name: 'Widget', slug: 'widget', description: '',
  status: 'active', has_variants: false, created_at: '2026-07-29T00:00:00Z', updated_at: '2026-07-29T00:00:00Z',
}

const sampleVariant: Variant = {
  id: 'v1', tenant_id: 't1', product_id: 'p1', name: 'Large', position: 0,
  created_at: '2026-07-29T00:00:00Z', updated_at: '2026-07-29T00:00:00Z',
}

const sampleSKU: SKU = {
  id: 's1', tenant_id: 't1', variant_id: 'v1', sku_code: 'WID-L',
  price_minor_units: 1999, currency_code: 'USD', is_virtual: false, status: 'active',
  created_at: '2026-07-29T00:00:00Z', updated_at: '2026-07-29T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listCategories', () => {
  it('returns categories', async () => {
    mockFetch([sampleCategory])
    const result = await api.listCategories(TOKEN)
    expect(result).toEqual([sampleCategory])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listCategories(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/categories`, expect.objectContaining({ method: 'GET' }))
  })
})

describe('createCategory', () => {
  it('sends name in body', async () => {
    mockFetch(sampleCategory, 201)
    await api.createCategory(TOKEN, { name: 'Electronics' })
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({ name: 'Electronics' })
  })

  it('throws on conflict', async () => {
    mockFetchError('CONFLICT', 'slug already exists', 409)
    await expect(api.createCategory(TOKEN, { name: 'Electronics' })).rejects.toThrow('slug already exists')
  })
})

describe('updateCategory', () => {
  it('calls correct endpoint', async () => {
    mockFetch(sampleCategory)
    await api.updateCategory(TOKEN, 'c1', { name: 'Electronics v2' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/categories/c1`, expect.objectContaining({ method: 'PATCH' }))
  })
})

describe('deleteCategory', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.deleteCategory(TOKEN, 'c1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/categories/c1`, expect.objectContaining({ method: 'DELETE' }))
  })
})

describe('listProducts', () => {
  it('returns products', async () => {
    mockFetch([sampleProduct])
    const result = await api.listProducts(TOKEN)
    expect(result).toEqual([sampleProduct])
  })
})

describe('createProduct', () => {
  it('sends fields in body', async () => {
    mockFetch(sampleProduct, 201)
    await api.createProduct(TOKEN, { name: 'Widget', status: 'draft' })
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({ name: 'Widget', status: 'draft' })
  })

  it('throws on duplicate slug', async () => {
    mockFetchError('CONFLICT', 'slug already exists', 409)
    await expect(api.createProduct(TOKEN, { name: 'Widget', status: 'draft' })).rejects.toThrow('slug already exists')
  })
})

describe('getProduct', () => {
  it('returns product detail with nested variants/skus', async () => {
    const detail: ProductDetail = { product: sampleProduct, variants: [{ ...sampleVariant, skus: [sampleSKU] }] }
    mockFetch(detail)
    const result = await api.getProduct(TOKEN, 'p1')
    expect(result.variants[0].skus?.[0].sku_code).toBe('WID-L')
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'product not found', 404)
    await expect(api.getProduct(TOKEN, 'missing')).rejects.toThrow('product not found')
  })
})

describe('updateProduct', () => {
  it('calls correct endpoint', async () => {
    mockFetch(sampleProduct)
    await api.updateProduct(TOKEN, 'p1', { name: 'Widget', status: 'active' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/products/p1`, expect.objectContaining({ method: 'PATCH' }))
  })
})

describe('deleteProduct', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.deleteProduct(TOKEN, 'p1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/products/p1`, expect.objectContaining({ method: 'DELETE' }))
  })
})

describe('createVariant', () => {
  it('posts to product variants endpoint', async () => {
    mockFetch(sampleVariant, 201)
    await api.createVariant(TOKEN, 'p1', { name: 'Large' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/products/p1/variants`, expect.objectContaining({ method: 'POST' }))
  })
})

describe('updateVariant', () => {
  it('calls correct endpoint', async () => {
    mockFetch(sampleVariant)
    await api.updateVariant(TOKEN, 'v1', { name: 'Large' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/variants/v1`, expect.objectContaining({ method: 'PATCH' }))
  })
})

describe('deleteVariant', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.deleteVariant(TOKEN, 'v1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/variants/v1`, expect.objectContaining({ method: 'DELETE' }))
  })
})

describe('createSKU', () => {
  it('posts to variant skus endpoint', async () => {
    mockFetch(sampleSKU, 201)
    await api.createSKU(TOKEN, 'v1', { sku_code: 'WID-L', price_minor_units: 1999, currency_code: 'USD', status: 'active' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/variants/v1/skus`, expect.objectContaining({ method: 'POST' }))
  })

  it('throws on duplicate sku code', async () => {
    mockFetchError('CONFLICT', 'sku code already exists', 409)
    await expect(api.createSKU(TOKEN, 'v1', { sku_code: 'WID-L', price_minor_units: 1999, currency_code: 'USD', status: 'active' }))
      .rejects.toThrow('sku code already exists')
  })
})

describe('updateSKU', () => {
  it('calls correct endpoint', async () => {
    mockFetch(sampleSKU)
    await api.updateSKU(TOKEN, 's1', { price_minor_units: 2499, currency_code: 'USD', status: 'active' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/skus/s1`, expect.objectContaining({ method: 'PATCH' }))
  })
})

describe('deleteSKU', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.deleteSKU(TOKEN, 's1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/skus/s1`, expect.objectContaining({ method: 'DELETE' }))
  })
})

const sampleMedia: Media = {
  id: 'm1', tenant_id: 't1', product_id: 'p1', url: 'https://api.example.com/uploads/t1/x.jpg',
  type: 'image', position: 0, alt_text: '', created_at: '2026-07-29T00:00:00Z',
}

describe('listProductMedia', () => {
  it('calls correct endpoint', async () => {
    mockFetch([sampleMedia])
    const result = await api.listProductMedia(TOKEN, 'p1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/products/p1/media`, expect.objectContaining({ method: 'GET' }))
    expect(result).toEqual([sampleMedia])
  })
})

describe('uploadProductMedia', () => {
  it('posts a FormData body without forcing a Content-Type header', async () => {
    mockFetch(sampleMedia, 201)
    const file = new File(['fake-bytes'], 'photo.jpg', { type: 'image/jpeg' })

    await api.uploadProductMedia(TOKEN, 'p1', file, 'Front view')

    expect(vi.mocked(fetch)).toHaveBeenCalledTimes(1)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe(`${BASE}/products/p1/media`)
    expect(options?.method).toBe('POST')
    expect(options?.body).toBeInstanceOf(FormData)
    expect((options?.body as FormData).get('file')).toBe(file)
    expect((options?.body as FormData).get('alt_text')).toBe('Front view')
    // the browser must set Content-Type itself (it includes the multipart
    // boundary) — request() always forces application/json, so this call
    // deliberately bypasses it.
    expect((options?.headers as Record<string, string>)['Content-Type']).toBeUndefined()
  })

  it('throws on an unsupported file type', async () => {
    mockFetchError('VALIDATION_ERROR', 'unsupported file type', 422)
    const file = new File(['bytes'], 'doc.pdf', { type: 'application/pdf' })

    await expect(api.uploadProductMedia(TOKEN, 'p1', file)).rejects.toThrow('unsupported file type')
  })
})

describe('updateMedia', () => {
  it('sends alt_text in body to the correct endpoint', async () => {
    mockFetch({ ...sampleMedia, alt_text: 'Front view' })
    await api.updateMedia(TOKEN, 'm1', 'Front view')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/media/m1`, expect.objectContaining({ method: 'PATCH' }))
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({ alt_text: 'Front view' })
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'media not found', 404)
    await expect(api.updateMedia(TOKEN, 'missing', 'Front view')).rejects.toThrow('media not found')
  })
})

describe('deleteMedia', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.deleteMedia(TOKEN, 'm1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/media/m1`, expect.objectContaining({ method: 'DELETE' }))
  })
})
