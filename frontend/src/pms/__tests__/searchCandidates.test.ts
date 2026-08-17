import { describe, it, expect, vi, beforeEach } from 'vitest'
import { searchSKUCandidates } from '../searchCandidates'
import type { SKU, Product, ProductDetail } from '../types'

vi.mock('../api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
}))

import * as pmsApi from '../api'

const TOKEN = 'test-token'

function sku(overrides: Partial<SKU> = {}): SKU {
  return {
    id: 'sku-1', tenant_id: 't1', variant_id: 'v1', sku_code: 'WID-L',
    price_minor_units: 1999, currency_code: 'USD', is_virtual: false, status: 'active',
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'p1', tenant_id: 't1', name: 'Widget', slug: 'widget', description: '',
    status: 'active', has_variants: false, created_at: '', updated_at: '',
    ...overrides,
  } as Product
}

function productDetail(p: Product, skus: SKU[]): ProductDetail {
  return {
    product: p,
    variants: [{ id: `v-${p.id}`, tenant_id: 't1', product_id: p.id, name: 'Default', position: 0, created_at: '', updated_at: '', skus }],
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('searchSKUCandidates', () => {
  it('returns an empty list when no product matches', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })

    const result = await searchSKUCandidates(TOKEN, 'nope')

    expect(result).toEqual([])
    expect(pmsApi.getProduct).not.toHaveBeenCalled()
  })

  it('flattens every matched product\'s SKUs into candidates, with the query passed through to ListProducts Search', async () => {
    const p1 = product({ id: 'p1', name: 'Widget' })
    const p2 = product({ id: 'p2', name: 'Widget Pro' })
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [p1, p2], total: 2 })
    vi.mocked(pmsApi.getProduct).mockImplementation(async (_token, id) =>
      id === 'p1' ? productDetail(p1, [sku({ id: 'sku-1', sku_code: 'WID-L' })])
        : productDetail(p2, [sku({ id: 'sku-2', sku_code: 'WID-PRO' })]))

    const result = await searchSKUCandidates(TOKEN, 'widg', 5)

    expect(pmsApi.listProducts).toHaveBeenCalledWith(TOKEN, { search: 'widg', perPage: 5 })
    expect(result).toEqual([
      { sku: sku({ id: 'sku-1', sku_code: 'WID-L' }), productId: 'p1', productName: 'Widget' },
      { sku: sku({ id: 'sku-2', sku_code: 'WID-PRO' }), productId: 'p2', productName: 'Widget Pro' },
    ])
  })
})
