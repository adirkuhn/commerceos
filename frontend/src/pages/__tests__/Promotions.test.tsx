// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import { act } from 'react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Promotions from '../Promotions'
import { renderAt } from '../../test/renderAt'
import type { Promotion, PromotionDetail } from '../../promotions/types'
import type { PromotionPerformanceReport } from '../../reports/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../promotions/api', () => ({
  listPromotions: vi.fn(),
  getPromotion: vi.fn(),
  createPromotion: vi.fn(),
  updatePromotion: vi.fn(),
  deletePromotion: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  resolveBarcode: vi.fn(),
  listProducts: vi.fn(),
  getProduct: vi.fn(),
}))

vi.mock('../../reports/api', () => ({
  getPromotionPerformanceReport: vi.fn(),
}))

import * as promotionsApi from '../../promotions/api'
import * as pmsApi from '../../pms/api'
import * as reportsApi from '../../reports/api'
import { acceptConfirm } from '../../test/confirm'

function promotion(overrides: Partial<Promotion> = {}): Promotion {
  return {
    id: 'promo-1', name: 'Summer Acai', type: 'percentage', starts_at: '2026-07-01T00:00:00Z',
    ends_at: '2026-07-31T00:00:00Z', status: 'active', created_at: '', updated_at: '',
    ...overrides,
  }
}

function promotionDetail(overrides: Partial<PromotionDetail> = {}): PromotionDetail {
  return { ...promotion(), lines: [], ...overrides }
}

function performanceReport(overrides: Partial<PromotionPerformanceReport> = {}): PromotionPerformanceReport {
  return {
    promotion_id: 'promo-1', promotion_name: 'Summer Acai', currency_code: 'EUR',
    before: { from: '2026-06-01T00:00:00Z', to: '2026-07-01T00:00:00Z', revenue_minor_units: 100000, units_sold: 100, order_count: 50 },
    during: { from: '2026-07-01T00:00:00Z', to: '2026-07-31T00:00:00Z', revenue_minor_units: 150000, units_sold: 120, order_count: 60 },
    revenue_change_percent: 50, units_sold_change_percent: 20,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(promotionsApi.listPromotions).mockResolvedValue([])
  vi.mocked(reportsApi.getPromotionPerformanceReport).mockResolvedValue(performanceReport())
})

describe('Promotions list', () => {
  it('shows an empty state when there are no promotions', async () => {
    renderAt(<Promotions />, '/promotions')

    expect(await screen.findByText('No promotions yet')).toBeInTheDocument()
  })

  it('lists promotions with type and status', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])

    renderAt(<Promotions />, '/promotions')

    expect(await screen.findByText('Summer Acai')).toBeInTheDocument()
    expect(screen.getByText('percentage')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([
      promotion(), promotion({ id: 'promo-2', name: 'Winter Sale' }),
    ])

    renderAt(<Promotions />, '/promotions')
    await screen.findByText('Summer Acai')

    await userEvent.type(screen.getByPlaceholderText('Search by name...'), 'Winter')

    expect(screen.queryByText('Summer Acai')).not.toBeInTheDocument()
    expect(screen.getByText('Winter Sale')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(promotionsApi.deletePromotion).toHaveBeenCalledWith('test-token', 'promo-1'))
    expect(screen.getByText('Add promotion')).toBeInTheDocument()
  })
})

describe('Promotion edit page', () => {
  it('opening Add Promotion shows a blank create form', async () => {
    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))

    expect(await screen.findByText('New promotion', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create promotion' })).toBeInTheDocument()
  })

  it('adding a SKU line resolves the code and shows product name', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: { id: 'sku-1', sku_code: 'ACAI-1L' } as never,
      units_per_scan: 1,
      product_name: 'Acai 1L',
    })

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.type(await screen.findByPlaceholderText('SKU code or barcode'), 'ACAI-1L')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))

    expect(await screen.findByText('Acai 1L')).toBeInTheDocument()
    expect(screen.getByText('ACAI-1L')).toBeInTheDocument()
  })

  it('adding a SKU line by searching the product name', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({
      products: [{
        id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Acai 1L',
        slug: 'acai-1l', description: '', status: 'active', has_variants: false, tags: [],
        created_at: '', updated_at: '',
      }],
      total: 1,
    })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: {
        id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Acai 1L',
        slug: 'acai-1l', description: '', status: 'active', has_variants: false, tags: [],
        created_at: '', updated_at: '',
      },
      variants: [{
        id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0,
        created_at: '', updated_at: '',
        skus: [{
          id: 'sku-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: 'ACAI-1L',
          price_minor_units: 999, currency_code: 'EUR', is_virtual: false, status: 'active',
          created_at: '', updated_at: '',
        }],
      }],
    } as never)

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Search product' }))
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Acai')
    await userEvent.click(await screen.findByRole('button', { name: 'Acai 1L' }))
    await userEvent.click(await screen.findByText('ACAI-1L'))

    expect(await screen.findByLabelText('Discount value')).toBeInTheDocument()
    expect(screen.getByText('Acai 1L')).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('Search products by name…')).not.toBeInTheDocument()
    expect(pmsApi.resolveBarcode).not.toHaveBeenCalled()
  })

  it('shows an error when the SKU code does not resolve', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.type(await screen.findByPlaceholderText('SKU code or barcode'), 'MISSING')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))

    expect(await screen.findByText('sku not found')).toBeInTheDocument()
  })

  it('creates a new promotion with name, type, dates, and lines', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: { id: 'sku-1', sku_code: 'ACAI-1L' } as never,
      units_per_scan: 1,
      product_name: 'Acai 1L',
    })
    vi.mocked(promotionsApi.createPromotion).mockResolvedValue({ promotion: promotionDetail({ id: 'promo-2', name: 'Winter Sale' }), warnings: [] })

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'Winter Sale')
    await userEvent.type(screen.getByLabelText('Starts at'), '2026-07-01T00:00')
    await userEvent.type(screen.getByPlaceholderText('SKU code or barcode'), 'ACAI-1L')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))
    await screen.findByText('Acai 1L')
    await userEvent.type(screen.getByLabelText('Discount value'), '1000')
    await userEvent.click(screen.getByRole('button', { name: 'Create promotion' }))

    await waitFor(() => expect(promotionsApi.createPromotion).toHaveBeenCalledWith('test-token', expect.objectContaining({
      name: 'Winter Sale',
      type: 'percentage',
      lines: [{ sku_id: 'sku-1', discount_type: 'percentage', discount_value: 1000 }],
    })))
    expect(await screen.findByText('Winter Sale')).toBeInTheDocument()
  })

  it('alerts with any warnings returned after creating a promotion', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: { id: 'sku-1', sku_code: 'ACAI-1L' } as never,
      units_per_scan: 1,
      product_name: 'Acai 1L',
    })
    vi.mocked(promotionsApi.createPromotion).mockResolvedValue({
      promotion: promotionDetail({ id: 'promo-2', name: 'Winter Sale' }),
      warnings: ['SKU ACAI-1L already has a manual sale price -- this promotion will override it while active'],
    })
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'Winter Sale')
    await userEvent.type(screen.getByLabelText('Starts at'), '2026-07-01T00:00')
    await userEvent.type(screen.getByPlaceholderText('SKU code or barcode'), 'ACAI-1L')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))
    await screen.findByText('Acai 1L')
    await userEvent.type(screen.getByLabelText('Discount value'), '1000')
    await userEvent.click(screen.getByRole('button', { name: 'Create promotion' }))

    await waitFor(() => expect(alertSpy).toHaveBeenCalledWith(expect.stringContaining('already has a manual sale price')))
    alertSpy.mockRestore()
  })

  it('rejects creating a promotion with no SKU lines', async () => {
    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'Empty Promo')

    expect(screen.getByRole('button', { name: 'Create promotion' })).toBeDisabled()
  })

  it('editing an existing promotion fetches and prefills its lines', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])
    vi.mocked(promotionsApi.getPromotion).mockResolvedValue(promotionDetail({
      lines: [{ id: 'line-1', sku_id: 'sku-1', sku_code: 'ACAI-1L', discount_type: 'percentage', discount_value: 1000 }],
    }))

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByText('Summer Acai'))

    await waitFor(() => expect(promotionsApi.getPromotion).toHaveBeenCalledWith('test-token', 'promo-1'))
    expect(await screen.findAllByText('ACAI-1L')).toHaveLength(2)
    expect(screen.getByDisplayValue('1000')).toBeInTheDocument()
  })

  it('shows the performance comparison for an existing promotion', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])
    vi.mocked(promotionsApi.getPromotion).mockResolvedValue(promotionDetail())

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByText('Summer Acai'))

    await waitFor(() => expect(reportsApi.getPromotionPerformanceReport).toHaveBeenCalledWith('test-token', 'promo-1'))
    expect(await screen.findByText('€1,500.00')).toBeInTheDocument()
    expect(screen.getByText('+50.0% vs before')).toBeInTheDocument()
  })

  it('shows an error if loading promotion performance fails', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])
    vi.mocked(promotionsApi.getPromotion).mockResolvedValue(promotionDetail())
    vi.mocked(reportsApi.getPromotionPerformanceReport).mockRejectedValue(new Error('performance fetch failed'))

    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByText('Summer Acai'))

    expect(await screen.findByText('performance fetch failed')).toBeInTheDocument()
  })

  it('does not show a performance card for a new (unsaved) promotion', async () => {
    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))

    expect(screen.queryByText('Performance')).not.toBeInTheDocument()
    expect(reportsApi.getPromotionPerformanceReport).not.toHaveBeenCalled()
  })

  it('Discard returns to the list without saving', async () => {
    renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: 'Add promotion' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'Draft Promo')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(promotionsApi.createPromotion).not.toHaveBeenCalled()
    expect(await screen.findByText('No promotions yet')).toBeInTheDocument()
  })
})

describe('Promotions URLs (refresh + Back)', () => {
  it('Add Promotion pushes /promotions/new and Back returns to the list', async () => {
    const { router } = renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByRole('button', { name: /Add promotion/ }))
    expect(router.state.location.pathname).toBe('/promotions/new')
    expect(screen.getByText('New promotion')).toBeInTheDocument()

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/promotions')
    expect(await screen.findByPlaceholderText('Search by name...')).toBeInTheDocument()
  })

  it('/promotions/:id loads that promotion on refresh', async () => {
    vi.mocked(promotionsApi.getPromotion).mockResolvedValue(promotionDetail())
    renderAt(<Promotions />, '/promotions', '/promotions/promo-1')
    await waitFor(() => expect(promotionsApi.getPromotion).toHaveBeenCalledWith('test-token', 'promo-1'))
    expect(await screen.findByDisplayValue('Summer Acai')).toBeInTheDocument()
  })

  it('clicking a row pushes /promotions/:id', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])
    vi.mocked(promotionsApi.getPromotion).mockResolvedValue(promotionDetail())
    const { router } = renderAt(<Promotions />, '/promotions')
    await userEvent.click(await screen.findByText('Summer Acai'))
    expect(router.state.location.pathname).toBe('/promotions/promo-1')
  })
})
