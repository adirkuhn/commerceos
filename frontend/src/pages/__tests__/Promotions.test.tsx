// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Promotions from '../Promotions'
import type { Promotion, PromotionDetail } from '../../promotions/types'

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
}))

import * as promotionsApi from '../../promotions/api'
import * as pmsApi from '../../pms/api'

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

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(promotionsApi.listPromotions).mockResolvedValue([])
})

describe('Promotions list', () => {
  it('shows an empty state when there are no promotions', async () => {
    render(<Promotions />)

    expect(await screen.findByText('No promotions yet')).toBeInTheDocument()
  })

  it('lists promotions with type and status', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])

    render(<Promotions />)

    expect(await screen.findByText('Summer Acai')).toBeInTheDocument()
    expect(screen.getByText('percentage')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([
      promotion(), promotion({ id: 'promo-2', name: 'Winter Sale' }),
    ])

    render(<Promotions />)
    await screen.findByText('Summer Acai')

    await userEvent.type(screen.getByPlaceholderText('Search by name...'), 'Winter')

    expect(screen.queryByText('Summer Acai')).not.toBeInTheDocument()
    expect(screen.getByText('Winter Sale')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<Promotions />)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(promotionsApi.deletePromotion).toHaveBeenCalledWith('test-token', 'promo-1'))
    expect(screen.getByText('Add Promotion')).toBeInTheDocument()
  })
})

describe('Promotion edit page', () => {
  it('opening Add Promotion shows a blank create form', async () => {
    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))

    expect(screen.getByText('New Promotion', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create Promotion' })).toBeInTheDocument()
  })

  it('adding a SKU line resolves the code and shows product name', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: { id: 'sku-1', sku_code: 'ACAI-1L' } as never,
      units_per_scan: 1,
      product_name: 'Acai 1L',
    })

    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))
    await userEvent.type(screen.getByPlaceholderText('SKU code'), 'ACAI-1L')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))

    expect(await screen.findByText('Acai 1L')).toBeInTheDocument()
    expect(screen.getByText('ACAI-1L')).toBeInTheDocument()
  })

  it('shows an error when the SKU code does not resolve', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))
    await userEvent.type(screen.getByPlaceholderText('SKU code'), 'MISSING')
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

    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Winter Sale')
    await userEvent.type(screen.getByLabelText('Starts at'), '2026-07-01T00:00')
    await userEvent.type(screen.getByPlaceholderText('SKU code'), 'ACAI-1L')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))
    await screen.findByText('Acai 1L')
    await userEvent.type(screen.getByLabelText('Discount value'), '1000')
    await userEvent.click(screen.getByRole('button', { name: 'Create Promotion' }))

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

    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Winter Sale')
    await userEvent.type(screen.getByLabelText('Starts at'), '2026-07-01T00:00')
    await userEvent.type(screen.getByPlaceholderText('SKU code'), 'ACAI-1L')
    await userEvent.click(screen.getByRole('button', { name: 'Add SKU' }))
    await screen.findByText('Acai 1L')
    await userEvent.type(screen.getByLabelText('Discount value'), '1000')
    await userEvent.click(screen.getByRole('button', { name: 'Create Promotion' }))

    await waitFor(() => expect(alertSpy).toHaveBeenCalledWith(expect.stringContaining('already has a manual sale price')))
    alertSpy.mockRestore()
  })

  it('rejects creating a promotion with no SKU lines', async () => {
    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Empty Promo')

    expect(screen.getByRole('button', { name: 'Create Promotion' })).toBeDisabled()
  })

  it('editing an existing promotion fetches and prefills its lines', async () => {
    vi.mocked(promotionsApi.listPromotions).mockResolvedValue([promotion()])
    vi.mocked(promotionsApi.getPromotion).mockResolvedValue(promotionDetail({
      lines: [{ id: 'line-1', sku_id: 'sku-1', sku_code: 'ACAI-1L', discount_type: 'percentage', discount_value: 1000 }],
    }))

    render(<Promotions />)
    await userEvent.click(await screen.findByText('Summer Acai'))

    await waitFor(() => expect(promotionsApi.getPromotion).toHaveBeenCalledWith('test-token', 'promo-1'))
    expect(await screen.findAllByText('ACAI-1L')).toHaveLength(2)
    expect(screen.getByDisplayValue('1000')).toBeInTheDocument()
  })

  it('Discard returns to the list without saving', async () => {
    render(<Promotions />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Promotion' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Draft Promo')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(promotionsApi.createPromotion).not.toHaveBeenCalled()
    expect(await screen.findByText('No promotions yet')).toBeInTheDocument()
  })
})
