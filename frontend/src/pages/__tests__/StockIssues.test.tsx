// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import StockIssues from '../StockIssues'
import type { SalesOrder, RetryStockDecrementResult } from '../../oms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  retryStockDecrement: vi.fn(),
}))

import * as omsApi from '../../oms/api'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    needs_stock_review: true,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
})

describe('StockIssues', () => {
  it('shows an empty state when there are no stock issues', async () => {
    render(<StockIssues />)

    expect(await screen.findByText("No stock issues — everything's resolved")).toBeInTheDocument()
  })

  it('lists orders flagged for stock review, requesting the needsStockReview filter', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    render(<StockIssues />)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', expect.objectContaining({ needsStockReview: true }))
  })

  it('retries the stock decrement and shows the resolved/still-pending summary', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    const result: RetryStockDecrementResult = { sales_order: salesOrder({ needs_stock_review: false }), resolved: 1, still_pending: 0 }
    vi.mocked(omsApi.retryStockDecrement).mockResolvedValue(result)

    render(<StockIssues />)
    await screen.findByText('#1001')

    await userEvent.click(screen.getByRole('button', { name: 'Retry stock' }))

    expect(omsApi.retryStockDecrement).toHaveBeenCalledWith('test-token', 'so-1')
    expect(await screen.findByText('1 resolved')).toBeInTheDocument()
    await waitFor(() => expect(omsApi.listSalesOrders).toHaveBeenCalledTimes(2))
  })

  it('reports still-pending line items after a partial retry', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.retryStockDecrement).mockResolvedValue({
      sales_order: salesOrder(), resolved: 0, still_pending: 1,
    })

    render(<StockIssues />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('button', { name: 'Retry stock' }))

    expect(await screen.findByText('0 resolved, 1 still pending')).toBeInTheDocument()
  })

  it('surfaces a retry error', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.retryStockDecrement).mockRejectedValue(new Error('order not stock reducing'))

    render(<StockIssues />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('button', { name: 'Retry stock' }))

    expect(await screen.findByText('order not stock reducing')).toBeInTheDocument()
  })
})
