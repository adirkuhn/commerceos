// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import i18next from 'i18next'
import StockIssues from '../StockIssues'
import { renderAt } from '../../test/renderAt'
import type { SalesOrder, SalesOrderLineItem, SalesOrderDetail, RetryStockDecrementResult, ForceAcceptStockResult } from '../../oms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  retryStockDecrement: vi.fn(),
  forceAcceptStock: vi.fn(),
  getSalesOrder: vi.fn(),
}))

import * as omsApi from '../../oms/api'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1', payment_status: 'paid',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    needs_stock_review: true,
    ...overrides,
  }
}

function lineItem(overrides: Partial<SalesOrderLineItem> = {}): SalesOrderLineItem {
  return {
    id: 'li-1', sku_id: 'sku-1', name: 'Widget A', sku_code: 'WID-A', barcode: '', quantity: 3,
    stock_status: 'pending', price_minor_units: 500, subtotal_minor_units: 1500,
    subtotal_tax_minor_units: 0, total_minor_units: 1500, total_tax_minor_units: 0,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
})

describe('StockIssues', () => {
  it('shows an empty state when there are no stock issues', async () => {
    renderAt(<StockIssues />, '/stock-issues')

    expect(await screen.findByText("No stock issues — everything's resolved")).toBeInTheDocument()
  })

  it('renders its texts in pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    try {
      renderAt(<StockIssues />, '/stock-issues')
      expect(await screen.findByText('Nenhum problema de estoque — tudo resolvido')).toBeInTheDocument()
    } finally {
      await i18next.changeLanguage('en')
    }
  })

  it('lists orders flagged for stock review, requesting the needsStockReview filter', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    renderAt(<StockIssues />, '/stock-issues')

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', expect.objectContaining({ needsStockReview: true }))
  })

  it('retries the stock decrement and shows the resolved/still-pending summary', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    const result: RetryStockDecrementResult = { sales_order: salesOrder({ needs_stock_review: false }), resolved: 1, still_pending: 0 }
    vi.mocked(omsApi.retryStockDecrement).mockResolvedValue(result)

    renderAt(<StockIssues />, '/stock-issues')
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

    renderAt(<StockIssues />, '/stock-issues')
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('button', { name: 'Retry stock' }))

    expect(await screen.findByText('0 resolved, 1 still pending')).toBeInTheDocument()
  })

  it('surfaces a retry error', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.retryStockDecrement).mockRejectedValue(new Error('order not stock reducing'))

    renderAt(<StockIssues />, '/stock-issues')
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('button', { name: 'Retry stock' }))

    expect(await screen.findByText('order not stock reducing')).toBeInTheDocument()
  })

  it('expanding a row fetches and shows which pending line item(s) are the problem', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    const detail: SalesOrderDetail = {
      sales_order: salesOrder(),
      line_items: [lineItem(), lineItem({ id: 'li-2', name: 'Widget B', sku_code: 'WID-B', stock_status: 'decremented' })],
    }
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail)

    renderAt(<StockIssues />, '/stock-issues')
    await userEvent.click(await screen.findByText('#1001'))

    expect(omsApi.getSalesOrder).toHaveBeenCalledWith('test-token', 'so-1')
    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('WID-A')).toBeInTheDocument()
    expect(screen.queryByText('Widget B')).not.toBeInTheDocument()
  })

  it('force-accepts stock, bypassing the check', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    const result: ForceAcceptStockResult = { sales_order: salesOrder({ needs_stock_review: false }), resolved: 1 }
    vi.mocked(omsApi.forceAcceptStock).mockResolvedValue(result)

    renderAt(<StockIssues />, '/stock-issues')
    await screen.findByText('#1001')

    await userEvent.click(screen.getByRole('button', { name: 'Force accept' }))

    expect(omsApi.forceAcceptStock).toHaveBeenCalledWith('test-token', 'so-1')
    expect(await screen.findByText('1 force-accepted')).toBeInTheDocument()
    await waitFor(() => expect(omsApi.listSalesOrders).toHaveBeenCalledTimes(2))
  })

  it('surfaces a force-accept error', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.forceAcceptStock).mockRejectedValue(new Error('order not found'))

    renderAt(<StockIssues />, '/stock-issues')
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('button', { name: 'Force accept' }))

    expect(await screen.findByText('order not found')).toBeInTheDocument()
  })
})

describe('StockIssues URLs', () => {
  it('requests the page in ?page=', async () => {
    renderAt(<StockIssues />, '/stock-issues', '/stock-issues?page=3')
    await waitFor(() => expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', expect.objectContaining({ page: 3 })))
  })
})
