// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { APIResponse, Category, Product, ProductDetail, Variant, SKU, Media, SKUCurrencyPrice } from '../types'

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
  status: 'active', has_variants: false, tags: [], created_at: '2026-07-29T00:00:00Z', updated_at: '2026-07-29T00:00:00Z',
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
  function mockProductListFetch(data: typeof sampleProduct[], total: number) {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data, total, error: null }),
    }))
  }

  it('returns products and total', async () => {
    mockProductListFetch([sampleProduct], 1)
    const result = await api.listProducts(TOKEN)
    expect(result).toEqual({ products: [sampleProduct], total: 1 })
  })

  it('calls the endpoint with no query string when no params are given', async () => {
    mockProductListFetch([], 0)
    await api.listProducts(TOKEN)
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/pms/products')
  })

  it('sends page, per_page and search as query params', async () => {
    mockProductListFetch([], 0)
    await api.listProducts(TOKEN, { page: 3, perPage: 50, search: 'shirt' })
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/pms/products?page=3&per_page=50&search=shirt')
  })

  it('sends the stock filter', async () => {
    mockProductListFetch([], 0)
    await api.listProducts(TOKEN, { page: 1, stock: 'low' })
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/pms/products?page=1&stock=low')
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listProducts(TOKEN)).rejects.toThrow('internal server error')
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
    await api.updateSKU(TOKEN, 's1', { sku_code: 'WID-L', price_minor_units: 2499, currency_code: 'USD', status: 'active' })
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

const sampleCurrencyPrice: SKUCurrencyPrice = {
  sku_id: 's1', currency_code: 'USD', price_minor_units: 2199, updated_at: '2026-09-08T00:00:00Z',
}

describe('listSKUCurrencyPrices', () => {
  it('calls the correct endpoint and returns the list', async () => {
    mockFetch([sampleCurrencyPrice])
    const result = await api.listSKUCurrencyPrices(TOKEN, 's1')
    expect(result).toEqual([sampleCurrencyPrice])
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/skus/s1/currency-prices`, expect.objectContaining({ method: 'GET' }))
  })
})

describe('upsertSKUCurrencyPrice', () => {
  it('PUTs the override and returns it', async () => {
    mockFetch(sampleCurrencyPrice)
    const result = await api.upsertSKUCurrencyPrice(TOKEN, 's1', { currency_code: 'USD', price_minor_units: 2199 })
    expect(result).toEqual(sampleCurrencyPrice)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      `${BASE}/skus/s1/currency-prices`,
      expect.objectContaining({ method: 'PUT', body: JSON.stringify({ currency_code: 'USD', price_minor_units: 2199 }) }),
    )
  })

  it('throws with the server error message on failure', async () => {
    mockFetchError('VALIDATION_ERROR', 'currency code must be 3 uppercase letters')
    await expect(api.upsertSKUCurrencyPrice(TOKEN, 's1', { currency_code: 'us', price_minor_units: 2199 })).rejects.toThrow('currency code must be 3 uppercase letters')
  })
})

describe('deleteSKUCurrencyPrice', () => {
  it('calls the correct endpoint', async () => {
    mockFetch(null)
    await api.deleteSKUCurrencyPrice(TOKEN, 's1', 'USD')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(`${BASE}/skus/s1/currency-prices/USD`, expect.objectContaining({ method: 'DELETE' }))
  })
})

describe('resolveBarcode', () => {
  it('calls the correct endpoint and returns the SKU plus units_per_scan/product_name', async () => {
    mockFetch({ sku: sampleSKU, units_per_scan: 12, product_name: 'Widget' })
    const result = await api.resolveBarcode(TOKEN, '012345')

    expect(result).toEqual({ sku: sampleSKU, units_per_scan: 12, product_name: 'Widget' })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      `${BASE}/skus/barcode-lookup/012345`,
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('url-encodes the scanned code', async () => {
    mockFetch({ sku: sampleSKU, units_per_scan: 1 })
    await api.resolveBarcode(TOKEN, 'A/B C')

    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      `${BASE}/skus/barcode-lookup/A%2FB%20C`,
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws when nothing matches', async () => {
    mockFetchError('NOT_FOUND', 'sku not found', 404)
    await expect(api.resolveBarcode(TOKEN, 'UNKNOWN')).rejects.toThrow('sku not found')
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

describe('packWeightPortions', () => {
  it('POSTs the package lines to the bulk SKU\'s weight-packages endpoint', async () => {
    const result = { lines: [{
      sku_id: 's1', sku_code: 'CARNE-SECA-380G', barcode: '2994721003804', price_minor_units: 11396,
      currency_code: 'EUR', weight_grams: 380, quantity: 10, created: true,
    }] }
    mockFetch(result)
    const input = { warehouse_id: 'w1', bulk_batch_id: 'b1', lines: [{ weight_grams: 380, quantity: 10 }] }

    await expect(api.packWeightPortions(TOKEN, 'sku-bulk-1', input)).resolves.toEqual(result)

    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe(`${BASE}/skus/sku-bulk-1/weight-packages`)
    expect(init?.method).toBe('POST')
    expect(JSON.parse(init?.body as string)).toEqual(input)
  })

  it('surfaces the server\'s error message', async () => {
    mockFetchError('INVALID_REQUEST', 'insufficient stock')
    await expect(api.packWeightPortions(TOKEN, 'sku-bulk-1', { warehouse_id: 'w1', lines: [{ weight_grams: 380, quantity: 1 }] }))
      .rejects.toThrow('insufficient stock')
  })
})
