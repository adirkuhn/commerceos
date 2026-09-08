// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Coupons from '../Coupons'
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

import * as couponsApi from '../../coupons/api'

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
    render(<Coupons />)

    expect(await screen.findByText('No coupons yet')).toBeInTheDocument()
  })

  it('lists coupons with discount and status', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])

    render(<Coupons />)

    expect(await screen.findByText('SUMMER10')).toBeInTheDocument()
    expect(screen.getByText('10.00%')).toBeInTheDocument()
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([
      coupon(), coupon({ id: 'coupon-2', code: 'WINTER20' }),
    ])

    render(<Coupons />)
    await screen.findByText('SUMMER10')

    await userEvent.type(screen.getByPlaceholderText('Search by code...'), 'WINTER')

    expect(screen.queryByText('SUMMER10')).not.toBeInTheDocument()
    expect(screen.getByText('WINTER20')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<Coupons />)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(couponsApi.deleteCoupon).toHaveBeenCalledWith('test-token', 'coupon-1'))
    expect(screen.getByText('Add Coupon')).toBeInTheDocument()
  })
})

describe('Coupon edit page', () => {
  it('opening Add Coupon shows a blank create form', async () => {
    render(<Coupons />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Coupon' }))

    expect(screen.getByText('New Coupon', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Code')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create Coupon' })).toBeInTheDocument()
  })

  it('creates a new coupon with code, type, and value', async () => {
    vi.mocked(couponsApi.createCoupon).mockResolvedValue(coupon({ id: 'coupon-2', code: 'WINTER20', discount_value: 2000 }))

    render(<Coupons />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Coupon' }))
    await userEvent.type(screen.getByLabelText('Code'), 'winter20')
    await userEvent.type(screen.getByLabelText('Percent off'), '20')
    await userEvent.click(screen.getByRole('button', { name: 'Create Coupon' }))

    await waitFor(() => expect(couponsApi.createCoupon).toHaveBeenCalledWith('test-token', expect.objectContaining({
      code: 'WINTER20',
      discount_type: 'percent',
      discount_value: 2000,
    })))
    expect(await screen.findByText('WINTER20')).toBeInTheDocument()
  })

  it('rejects creating a coupon with a zero discount value', async () => {
    render(<Coupons />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Coupon' }))
    await userEvent.type(screen.getByLabelText('Code'), 'EMPTY')
    await userEvent.type(screen.getByLabelText('Percent off'), '0')
    await userEvent.click(screen.getByRole('button', { name: 'Create Coupon' }))

    expect(await screen.findByText('Discount value must be a positive number')).toBeInTheDocument()
    expect(couponsApi.createCoupon).not.toHaveBeenCalled()
  })

  it('editing an existing coupon prefills its fields', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon({ usage_limit: 100, times_used: 3 })])

    render(<Coupons />)
    await userEvent.click(await screen.findByText('SUMMER10'))

    expect(await screen.findByDisplayValue('SUMMER10')).toBeInTheDocument()
    expect(screen.getByDisplayValue('10')).toBeInTheDocument()
    expect(screen.getByText('Used 3 times')).toBeInTheDocument()
  })

  it('shows no currency-overrides editor for a Percent coupon', async () => {
    vi.mocked(couponsApi.listCoupons).mockResolvedValue([coupon()])

    render(<Coupons />)
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

    render(<Coupons />)
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

    render(<Coupons />)
    await userEvent.click(await screen.findByText('SUMMER10'))
    await screen.findByText('USD: 6.00 off')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(couponsApi.deleteCouponCurrencyAmount).toHaveBeenCalledWith('test-token', 'coupon-1', 'USD'))
    expect(screen.queryByText('USD: 6.00 off')).not.toBeInTheDocument()
  })

  it('Discard returns to the list without saving', async () => {
    render(<Coupons />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Coupon' }))
    await userEvent.type(screen.getByLabelText('Code'), 'DRAFT')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(couponsApi.createCoupon).not.toHaveBeenCalled()
    expect(await screen.findByText('No coupons yet')).toBeInTheDocument()
  })
})
