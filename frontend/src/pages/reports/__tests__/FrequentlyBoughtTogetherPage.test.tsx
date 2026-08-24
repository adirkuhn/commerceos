// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import FrequentlyBoughtTogetherPage from '../FrequentlyBoughtTogetherPage'
import type { FrequentlyBoughtTogetherReport, ProductRecommendationsReport } from '../../../reports/types'
import type { Product, ProductDetail, SKU } from '../../../pms/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getFrequentlyBoughtTogetherReport: vi.fn(),
  getProductRecommendations: vi.fn(),
}))

vi.mock('../../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'
import * as pmsApi from '../../../pms/api'

function fbtReport(overrides: Partial<FrequentlyBoughtTogetherReport> = {}): FrequentlyBoughtTogetherReport {
  return {
    total_orders: 20,
    rows: [
      {
        items: [
          { sku_id: 'sku-a', sku_code: 'ACAI-1L', product_name: 'Acai 1L' },
          { sku_id: 'sku-b', sku_code: 'GRANOLA-500G', product_name: 'Granola 500g' },
        ],
        support: 8, support_percent: 40,
      },
    ],
    ...overrides,
  }
}

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'product-1', tenant_id: 't1', category_id: null, name: 'Acai 1L', slug: 'acai-1l',
    description: '', status: 'active', has_variants: false,
    ...overrides,
  } as Product
}

function singleSkuDetail(): ProductDetail {
  return {
    product: product(),
    variants: [{
      id: 'variant-1', tenant_id: 't1', product_id: 'product-1', name: 'Default', position: 0,
      created_at: '', updated_at: '',
      skus: [{ id: 'sku-a', tenant_id: 't1', variant_id: 'variant-1', sku_code: 'ACAI-1L', price_minor_units: 500, currency_code: 'EUR', is_virtual: false, tracks_expiry: false } as SKU],
    }],
  }
}

function recommendations(overrides: Partial<ProductRecommendationsReport> = {}): ProductRecommendationsReport {
  return {
    sku_id: 'sku-a', total_orders: 20,
    rows: [{ sku_id: 'sku-b', sku_code: 'GRANOLA-500G', product_name: 'Granola 500g', support: 8 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getFrequentlyBoughtTogetherReport).mockResolvedValue(fbtReport())
  vi.mocked(reportsApi.getProductRecommendations).mockResolvedValue(recommendations())
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product()], total: 1 })
  vi.mocked(pmsApi.getProduct).mockResolvedValue(singleSkuDetail())
})

describe('FrequentlyBoughtTogetherPage', () => {
  it('shows the tenant-wide frequently bought together table', async () => {
    render(<FrequentlyBoughtTogetherPage />)

    expect(await screen.findByText('Acai 1L + Granola 500g')).toBeInTheDocument()
    const row = screen.getByTestId('fbt-row')
    expect(row).toHaveTextContent('8')
    expect(row).toHaveTextContent('40.0%')
  })

  it('shows an empty state when no SKU sets co-occurred frequently', async () => {
    vi.mocked(reportsApi.getFrequentlyBoughtTogetherReport).mockResolvedValue(fbtReport({ rows: [] }))

    render(<FrequentlyBoughtTogetherPage />)

    expect(await screen.findByText('No frequently co-occurring SKU sets in this period')).toBeInTheDocument()
  })

  it('searches a product, auto-selects its only SKU, and shows recommendations', async () => {
    const user = userEvent.setup()
    render(<FrequentlyBoughtTogetherPage />)
    await screen.findByText('Acai 1L + Granola 500g')

    await user.type(screen.getByPlaceholderText('Search products…'), 'acai')
    const result = await screen.findByRole('button', { name: 'Acai 1L' })
    await user.click(result)

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledWith('test-token', 'product-1'))
    const recs = await screen.findByTestId('product-recommendations')
    expect(recs).toHaveTextContent('Granola 500g')
    expect(recs).toHaveTextContent('support 8')
    expect(reportsApi.getProductRecommendations).toHaveBeenCalledWith('test-token', 'sku-a', expect.any(String), expect.any(String))
  })

  it('shows a SKU picker when the selected product has multiple variants', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [
        { id: 'variant-1', tenant_id: 't1', product_id: 'product-1', name: 'Small', position: 0, created_at: '', updated_at: '', skus: [{ id: 'sku-a', tenant_id: 't1', variant_id: 'variant-1', sku_code: 'ACAI-1L-S', price_minor_units: 400, currency_code: 'EUR', is_virtual: false, tracks_expiry: false } as SKU] },
        { id: 'variant-2', tenant_id: 't1', product_id: 'product-1', name: 'Large', position: 1, created_at: '', updated_at: '', skus: [{ id: 'sku-c', tenant_id: 't1', variant_id: 'variant-2', sku_code: 'ACAI-1L-L', price_minor_units: 600, currency_code: 'EUR', is_virtual: false, tracks_expiry: false } as SKU] },
      ],
    })
    const user = userEvent.setup()
    render(<FrequentlyBoughtTogetherPage />)
    await screen.findByText('Acai 1L + Granola 500g')

    await user.type(screen.getByPlaceholderText('Search products…'), 'acai')
    await user.click(await screen.findByRole('button', { name: 'Acai 1L' }))

    expect(await screen.findByText(/ACAI-1L-S/)).toBeInTheDocument()
    expect(screen.getByText(/ACAI-1L-L/)).toBeInTheDocument()
    expect(reportsApi.getProductRecommendations).not.toHaveBeenCalled()
  })
})
