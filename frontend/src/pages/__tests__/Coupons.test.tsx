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

  it('Discard returns to the list without saving', async () => {
    render(<Coupons />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Coupon' }))
    await userEvent.type(screen.getByLabelText('Code'), 'DRAFT')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(couponsApi.createCoupon).not.toHaveBeenCalled()
    expect(await screen.findByText('No coupons yet')).toBeInTheDocument()
  })
})
