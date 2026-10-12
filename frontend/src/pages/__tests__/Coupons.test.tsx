// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import { act } from 'react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import i18next from 'i18next'
import Coupons from '../Coupons'
import { renderAt } from '../../test/renderAt'
import type { Coupon } from '../../coupons/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../coupons/api', () => ({
  listCoupons: vi.fn(),
  createCoupon: vi.fn(),
  updateCoupon: vi.fn(),
  deleteCoupon: vi.fn(),
  listCouponCurrencyAmounts: vi.fn(() => Promise.resolve([])),
  upsertCouponCurrencyAmount: vi.fn(),
  deleteCouponCurrencyAmount: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(() => Promise.resolve({ default_currency: 'EUR' })),
}))

import * as couponsApi from '../../coupons/api'
import { AmountVisibilityProvider } from '../../format/amountVisibility'
import { formatMoney } from '../../format/money'
import { acceptConfirm } from '../../test/confirm'

function coupon(overrides: Partial<Coupon> = {}): Coupon {
  return {
    id: 'coupon-1', code: 'SUMMER10', discount_type: 'percent', discount_value: 1000,
    min_subtotal_minor_units: null, usage_limit: null, times_used: 0, expires_at: null,
    active: true, created_at: '', updated_at: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(couponsApi.listCoupons).mockResolvedValue([])
})

describe('Coupons list', () => {
  it('shows an empty state when there are no coupons', async () => {
    renderAt(<Coupons />, '/coupons')

    expect(await screen.findByText('No coupons yet')).toBeInTheDocument()
  })

  it('lists coupons with discount and status', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])

    renderAt(<Coupons />, '/coupons')

    expect(await screen.findByText('SUMMER10')).toBeInTheDocument()
    expect(screen.getByText('10.00%')).toBeInTheDocument()
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([
      coupon(), coupon({ id: 'coupon-2', code: 'WINTER20' }),
    ])

    renderAt(<Coupons />, '/coupons')
    await screen.findByText('SUMMER10')

    await userEvent.type(screen.getByPlaceholderText('Search by code...'), 'WINTER')

    expect(screen.queryByText('SUMMER10')).not.toBeInTheDocument()
    expect(screen.getByText('WINTER20')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])

    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(couponsApi.deleteCoupon).toHaveBeenCalledWith('test-token', 'coupon-1'))
    expect(screen.getByText('Add coupon')).toBeInTheDocument()
  })
})

describe('Coupon edit page', () => {
  it('opening Add Coupon shows a blank create form', async () => {
    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByRole('button', { name: 'Add coupon' }))

    expect(await screen.findByText('New coupon', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Code')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create coupon' })).toBeInTheDocument()
  })

  it('creates a new coupon with code, type, and value', async () => {
    vi.mocked(couponsApi.createCoupon).mockResolvedValue(coupon({ id: 'coupon-2', code: 'WINTER20', discount_value: 2000 }))

    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByRole('button', { name: 'Add coupon' }))
    await userEvent.type(await screen.findByLabelText('Code'), 'winter20')
    await userEvent.type(screen.getByLabelText('Percent off'), '20')
    await userEvent.click(screen.getByRole('button', { name: 'Create coupon' }))

    await waitFor(() => expect(couponsApi.createCoupon).toHaveBeenCalledWith('test-token', expect.objectContaining({
      code: 'WINTER20',
      discount_type: 'percent',
      discount_value: 2000,
    })))
    expect(await screen.findByText('WINTER20')).toBeInTheDocument()
  })

  it('rejects creating a coupon with a zero discount value', async () => {
    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByRole('button', { name: 'Add coupon' }))
    await userEvent.type(await screen.findByLabelText('Code'), 'EMPTY')
    await userEvent.type(screen.getByLabelText('Percent off'), '0')
    await userEvent.click(screen.getByRole('button', { name: 'Create coupon' }))

    expect(await screen.findByText('Discount value must be a positive number')).toBeInTheDocument()
    expect(couponsApi.createCoupon).not.toHaveBeenCalled()
  })

  it('editing an existing coupon prefills its fields', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon({ usage_limit: 100, times_used: 3 })])

    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByText('SUMMER10'))

    expect(await screen.findByDisplayValue('SUMMER10')).toBeInTheDocument()
    expect(screen.getByDisplayValue('10')).toBeInTheDocument()
    expect(screen.getByText('Used 3 times')).toBeInTheDocument()
  })

  it('shows no currency-overrides editor for a Percent coupon', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])

    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByText('SUMMER10'))
    await screen.findByDisplayValue('SUMMER10')

    expect(screen.queryByText('Currency overrides')).not.toBeInTheDocument()
  })

  it('a Fixed coupon shows currency overrides, loads existing ones, and can add one', async () => {
    const fixedCoupon = coupon({ discount_type: 'fixed', discount_value: 500 })
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([fixedCoupon])
    vi.mocked(couponsApi.listCouponCurrencyAmounts).mockResolvedValue([
      { coupon_id: 'coupon-1', currency_code: 'USD', amount_minor_units: 600 },
    ])
    vi.mocked(couponsApi.upsertCouponCurrencyAmount).mockResolvedValue({ coupon_id: 'coupon-1', currency_code: 'GBP', amount_minor_units: 450 })

    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByText('SUMMER10'))
    await screen.findByDisplayValue('SUMMER10')

    expect(await screen.findByText('USD: 6.00 off')).toBeInTheDocument()
    expect(couponsApi.listCouponCurrencyAmounts).toHaveBeenCalledWith('test-token', 'coupon-1')

    await userEvent.selectOptions(screen.getByLabelText('Currency'), 'GBP')
    await userEvent.type(screen.getByLabelText('Override amount'), '4.50')
    await userEvent.click(screen.getByRole('button', { name: 'Add / update' }))

    await waitFor(() => expect(couponsApi.upsertCouponCurrencyAmount).toHaveBeenCalledWith('test-token', 'coupon-1', {
      currency_code: 'GBP', amount_minor_units: 450,
    }))
  })

  it('can delete a coupon currency override', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon({ discount_type: 'fixed', discount_value: 500 })])
    vi.mocked(couponsApi.listCouponCurrencyAmounts).mockResolvedValue([
      { coupon_id: 'coupon-1', currency_code: 'USD', amount_minor_units: 600 },
    ])
    vi.mocked(couponsApi.deleteCouponCurrencyAmount).mockResolvedValue(null)

    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByText('SUMMER10'))
    await screen.findByText('USD: 6.00 off')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(couponsApi.deleteCouponCurrencyAmount).toHaveBeenCalledWith('test-token', 'coupon-1', 'USD'))
    expect(screen.queryByText('USD: 6.00 off')).not.toBeInTheDocument()
  })

  it('Discard returns to the list without saving', async () => {
    renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByRole('button', { name: 'Add coupon' }))
    await userEvent.type(await screen.findByLabelText('Code'), 'DRAFT')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(couponsApi.createCoupon).not.toHaveBeenCalled()
    expect(await screen.findByText('No coupons yet')).toBeInTheDocument()
  })
})

describe('Coupons URLs (refresh + Back)', () => {
  it('Add Coupon pushes /coupons/new and Back returns to the list', async () => {
    const { router } = renderAt(<Coupons />, '/coupons')
    await userEvent.click(await screen.findByRole('button', { name: /Add coupon/ }))
    expect(router.state.location.pathname).toBe('/coupons/new')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/coupons')
    expect(await screen.findByPlaceholderText('Search by code...')).toBeInTheDocument()
  })

  it('/coupons/:id opens that coupon for editing on load', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])
    renderAt(<Coupons />, '/coupons', '/coupons/coupon-1')
    expect(await screen.findByDisplayValue('SUMMER10')).toBeInTheDocument()
  })

  it('/coupons/:id for an unknown id shows not-found instead of a blank create form', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])
    renderAt(<Coupons />, '/coupons', '/coupons/nope')
    expect(await screen.findByText('Coupon not found')).toBeInTheDocument()
  })

  it('page and search live in the query string', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon({ code: 'ALPHA' }), coupon({ id: 'c2', code: 'BETA' })])
    const { router } = renderAt(<Coupons />, '/coupons', '/coupons?q=bet')
    expect(await screen.findByText('BETA')).toBeInTheDocument()
    expect(screen.queryByText('ALPHA')).not.toBeInTheDocument()
    await userEvent.clear(screen.getByPlaceholderText('Search by code...'))
    await userEvent.type(screen.getByPlaceholderText('Search by code...'), 'al')
    expect(router.state.location.search).toBe('?q=al')
  })
})

describe('in Brazilian Portuguese', () => {
  afterEach(async () => { await i18next.changeLanguage('en') })

  it('shows the empty state in pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    renderAt(<Coupons />, '/coupons')

    expect(await screen.findByText('Nenhum cupom ainda')).toBeInTheDocument()
  })
})

describe('Coupons list amounts', () => {
  it('shows a fixed discount in the shop currency, hidden until clicked', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon({ discount_type: 'fixed', discount_value: 500 })])
    renderAt(<AmountVisibilityProvider><Coupons /></AmountVisibilityProvider>, '/coupons')

    const reveal = await screen.findByRole('button', { name: 'Show amount' })
    expect(screen.queryByText(formatMoney(500, 'EUR'))).not.toBeInTheDocument()
    await waitFor(() => expect(reveal.textContent).toContain('€'))

    await userEvent.click(reveal)

    expect(screen.getByText(formatMoney(500, 'EUR'))).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Show amount' })).not.toBeInTheDocument()
  })

  it('keeps percentages visible', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])
    renderAt(<AmountVisibilityProvider><Coupons /></AmountVisibilityProvider>, '/coupons')

    expect(await screen.findByText('10.00%')).toBeInTheDocument()
  })
})
