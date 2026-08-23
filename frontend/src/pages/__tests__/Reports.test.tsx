// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Reports from '../Reports'
import type { SalesSummaryReport, ProductPerformanceReport, ProductPerformanceMatrix, CustomerPerformanceReport } from '../../reports/types'
import type { DashboardStatsReport } from '../../dashboard/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../reports/api', () => ({
  getSalesSummaryReport: vi.fn(),
  getProductPerformanceReport: vi.fn(),
  getProductPerformanceMatrix: vi.fn(),
  getCustomerPerformanceReport: vi.fn(),
}))

vi.mock('../../dashboard/api', () => ({
  getDashboardStats: vi.fn(),
}))

import * as reportsApi from '../../reports/api'
import * as dashboardApi from '../../dashboard/api'

// jsdom doesn't implement ResizeObserver, which recharts' ResponsiveContainer requires to mount.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

function salesSummary(overrides: Partial<SalesSummaryReport> = {}): SalesSummaryReport {
  return {
    currency_code: 'EUR',
    current: { revenue_minor_units: 118000, order_count: 112, units_sold: 217, average_order_value_minor_units: 1053 },
    previous: { revenue_minor_units: 100000, order_count: 100, units_sold: 178, average_order_value_minor_units: 1000 },
    revenue_change_percent: 18.4, order_count_change_percent: 12.1, units_sold_change_percent: 21.9, aov_change_percent: 5.3,
    ...overrides,
  }
}

function productPerformance(overrides: Partial<ProductPerformanceReport> = {}): ProductPerformanceReport {
  return {
    currency_code: 'EUR',
    rows: [{
      sku_id: 'sku-1', sku_code: 'ACAI-1L', product_name: 'Acai 1L', quantity_sold: 234, revenue_minor_units: 585000,
      order_count: 90, first_sale_at: '2026-08-01T00:00:00Z', last_sale_at: '2026-08-15T00:00:00Z',
      cost_price_minor_units: 300, profit_minor_units: 515000, margin_basis_points: 8800,
    }],
    ...overrides,
  }
}

function productPerformanceMatrix(overrides: Partial<ProductPerformanceMatrix> = {}): ProductPerformanceMatrix {
  return {
    currency_code: 'EUR', median_revenue_minor_units: 10000, median_margin_basis_points: 3000,
    rows: [
      {
        sku_id: 'sku-1', sku_code: 'WINNER-1', product_name: 'Winner Product', quantity_sold: 100, revenue_minor_units: 50000,
        order_count: 40, first_sale_at: '2026-08-01T00:00:00Z', last_sale_at: '2026-08-15T00:00:00Z',
        cost_price_minor_units: 200, profit_minor_units: 30000, margin_basis_points: 6000, quadrant: 'winners',
      },
      {
        sku_id: 'sku-2', sku_code: 'PROBLEM-1', product_name: 'Problem Product', quantity_sold: 5, revenue_minor_units: 500,
        order_count: 2, first_sale_at: '2026-08-01T00:00:00Z', last_sale_at: '2026-08-02T00:00:00Z',
        cost_price_minor_units: 90, profit_minor_units: 50, margin_basis_points: 1000, quadrant: 'problems',
      },
    ],
    ...overrides,
  }
}

function customerPerformance(overrides: Partial<CustomerPerformanceReport> = {}): CustomerPerformanceReport {
  return {
    currency_code: 'EUR',
    rows: [{
      customer_id: 'cust-1', customer_name: 'Jane Doe', order_count: 4, revenue_minor_units: 42000,
      average_order_value_minor_units: 10500, first_order_at: '2026-07-01T00:00:00Z', last_order_at: '2026-08-15T00:00:00Z',
    }],
    ...overrides,
  }
}

function dashboardStats(overrides: Partial<DashboardStatsReport> = {}): DashboardStatsReport {
  return {
    total_stock_units: 0, currency_code: 'EUR', revenue_this_month_minor_units: 0,
    stock_flow_last_7_days: [], sales_by_channel_this_month: [{ source: 'woocommerce', total_minor_units: 31500 }],
    top_products_this_month: [],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.mocked(reportsApi.getSalesSummaryReport).mockResolvedValue(salesSummary())
  vi.mocked(reportsApi.getProductPerformanceReport).mockResolvedValue(productPerformance())
  vi.mocked(reportsApi.getProductPerformanceMatrix).mockResolvedValue(productPerformanceMatrix())
  vi.mocked(reportsApi.getCustomerPerformanceReport).mockResolvedValue(customerPerformance())
  vi.mocked(dashboardApi.getDashboardStats).mockResolvedValue(dashboardStats())
})

describe('Reports page', () => {
  it('shows real revenue and order KPIs with period-over-period change', async () => {
    render(<Reports />)

    expect(await screen.findByText('€1,180.00')).toBeInTheDocument()
    expect(screen.getByText('+18.4%')).toBeInTheDocument()
    expect(screen.getByText('112')).toBeInTheDocument()
    expect(screen.getByText('+12.1%')).toBeInTheDocument()
  })

  it('shows real top products with margin', async () => {
    render(<Reports />)

    expect(await screen.findByText('Acai 1L')).toBeInTheDocument()
    expect(screen.getByText('88.0%')).toBeInTheDocument()
  })

  it('shows real revenue by channel, labeled as this month', async () => {
    render(<Reports />)

    expect(await screen.findByText('woocommerce')).toBeInTheDocument()
    expect(screen.getByText(/this month/i)).toBeInTheDocument()
  })

  it('refetches with a wider range when switching period', async () => {
    render(<Reports />)
    await screen.findByText('€1,180.00')

    await userEvent.click(screen.getByRole('button', { name: '90D' }))

    await waitFor(() => expect(reportsApi.getSalesSummaryReport).toHaveBeenCalledTimes(2))
    const [, from90, to90] = vi.mocked(reportsApi.getSalesSummaryReport).mock.calls[1]
    const [, from7] = vi.mocked(reportsApi.getSalesSummaryReport).mock.calls[0]
    expect(new Date(from90).getTime()).toBeLessThan(new Date(from7).getTime())
    expect(to90).toBeTruthy()
  })

  it('shows a dash instead of a change badge when the previous baseline is zero', async () => {
    vi.mocked(reportsApi.getSalesSummaryReport).mockResolvedValue(salesSummary({ revenue_change_percent: null }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.queryByText('+18.4%')).not.toBeInTheDocument()
  })

  it('shows an error message when a report fails to load', async () => {
    vi.mocked(reportsApi.getSalesSummaryReport).mockRejectedValue(new Error('internal server error'))

    render(<Reports />)

    expect(await screen.findByText('internal server error')).toBeInTheDocument()
  })

  it('places each product in its quadrant', async () => {
    render(<Reports />)

    const winners = await screen.findByTestId('quadrant-winners')
    expect(within(winners).getByText('Winner Product')).toBeInTheDocument()
    const problems = screen.getByTestId('quadrant-problems')
    expect(within(problems).getByText('Problem Product')).toBeInTheDocument()
    expect(within(winners).queryByText('Problem Product')).not.toBeInTheDocument()
  })

  it('shows the median revenue and margin the quadrants are split around', async () => {
    render(<Reports />)

    await screen.findByText('Winner Product')
    expect(screen.getByText(/€100\.00/)).toBeInTheDocument()
    expect(screen.getByText(/30\.0%/)).toBeInTheDocument()
  })

  it('shows an empty state when no SKU has a known margin for the period', async () => {
    vi.mocked(reportsApi.getProductPerformanceMatrix).mockResolvedValue(productPerformanceMatrix({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No SKUs with a known margin in this period')).toBeInTheDocument()
  })

  it('shows real top customers by revenue', async () => {
    render(<Reports />)

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('€420.00')).toBeInTheDocument()
    expect(screen.getByText(/4 orders/)).toBeInTheDocument()
  })

  it('shows an empty state when no customer ordered in the period', async () => {
    vi.mocked(reportsApi.getCustomerPerformanceReport).mockResolvedValue(customerPerformance({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No customer orders in this period')).toBeInTheDocument()
  })
})
