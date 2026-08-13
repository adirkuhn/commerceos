// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import SalesOrders from '../SalesOrders'
import type { SalesOrder, SalesOrderDetail } from '../../oms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  getSalesOrder: vi.fn(),
}))

import * as omsApi from '../../oms/api'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'pos-desktop', external_order_id: 'ext-1',
    number: '1001', order_key: 'wc_order_abc', status: 'completed', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
})

describe('Sales orders list', () => {
  it('shows an empty state when there are no sales orders', async () => {
    render(<SalesOrders />)

    expect(await screen.findByText('No sales orders yet')).toBeInTheDocument()
  })

  it('lists sales orders with customer, status, and total', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    render(<SalesOrders />)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('completed')).toBeInTheDocument()
    expect(screen.getByText('$50.00')).toBeInTheDocument()
  })

  it('paginates using the server page param', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 25 })

    render(<SalesOrders />)
    await screen.findByText('#1001')

    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', { page: 2, perPage: 20 })
  })
})

describe('Sales order detail page', () => {
  function detail(overrides: Partial<SalesOrderDetail> = {}): SalesOrderDetail {
    return {
      sales_order: salesOrder(),
      line_items: [{
        id: 'li-1', sku_id: 'sku-1', name: 'Blue Widget', sku_code: 'WID-BLU',
        quantity: 2, price_minor_units: 2000, subtotal_minor_units: 4000, subtotal_tax_minor_units: 0,
        total_minor_units: 4000, total_tax_minor_units: 0,
      }],
      ...overrides,
    }
  }

  it('opens the detail page and shows line items when clicking a row', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Blue Widget')).toBeInTheDocument()
    expect(screen.getByText('WID-BLU')).toBeInTheDocument()
    expect(omsApi.getSalesOrder).toHaveBeenCalledWith('test-token', 'so-1')
  })

  it('Back returns to the list', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByRole('button', { name: /Sales Orders/ }))

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.queryByText('Blue Widget')).not.toBeInTheDocument()
  })

  it('shows an error if fetching the detail fails', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockRejectedValue(new Error('not found'))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('not found')).toBeInTheDocument()
  })
})
