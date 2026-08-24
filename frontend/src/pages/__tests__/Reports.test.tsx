// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Reports from '../Reports'
import type { SalesSummaryReport, ProductPerformanceReport, ProductPerformanceMatrix, CustomerPerformanceReport, CategoryPerformanceReport, SlowMovingProductsReport, ProfitabilityReport, InventorySalesReport, BusinessInsightsReport, SalesTrendsReport, ProductLifecycleReport, CustomerCohortsReport, CustomerRFMReport, FrequentlyBoughtTogetherReport } from '../../reports/types'
import type { DashboardStatsReport } from '../../dashboard/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../reports/api', () => ({
  getSalesSummaryReport: vi.fn(),
  getProductPerformanceReport: vi.fn(),
  getProductPerformanceMatrix: vi.fn(),
  getCustomerPerformanceReport: vi.fn(),
  getCategoryPerformanceReport: vi.fn(),
  getSlowMovingProductsReport: vi.fn(),
  getProfitabilityReport: vi.fn(),
  getInventorySalesReport: vi.fn(),
  getBusinessInsights: vi.fn(),
  getSalesTrendsReport: vi.fn(),
  getProductLifecycleReport: vi.fn(),
  getCustomerCohortsReport: vi.fn(),
  getCustomerRFMReport: vi.fn(),
  getFrequentlyBoughtTogetherReport: vi.fn(),
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

function categoryPerformance(overrides: Partial<CategoryPerformanceReport> = {}): CategoryPerformanceReport {
  return {
    currency_code: 'EUR',
    rows: [
      { category_id: 'cat-1', category_name: 'Beverages', quantity_sold: 50, revenue_minor_units: 40000, order_count: 20 },
      { category_id: null, category_name: 'Uncategorized', quantity_sold: 3, revenue_minor_units: 1500, order_count: 2 },
    ],
    ...overrides,
  }
}

function slowMovingProducts(overrides: Partial<SlowMovingProductsReport> = {}): SlowMovingProductsReport {
  return {
    rows: [{ sku_id: 'sku-slow', sku_code: 'SLOW-1', product_name: 'Slow Widget', stock_on_hand: 42, quantity_sold_in_period: 0 }],
    ...overrides,
  }
}

function profitability(overrides: Partial<ProfitabilityReport> = {}): ProfitabilityReport {
  return {
    currency_code: 'EUR',
    summary: {
      revenue_minor_units: 220000, cost_known_revenue_minor_units: 180000, cost_minor_units: 100000,
      profit_minor_units: 80000, margin_basis_points: 4444,
    },
    ...overrides,
  }
}

function inventorySales(overrides: Partial<InventorySalesReport> = {}): InventorySalesReport {
  return {
    rows: [{ sku_id: 'sku-turn', sku_code: 'TURN-1', product_name: 'Turning Widget', stock_on_hand: 40, quantity_sold_in_period: 10, turnover_basis_points: 2500 }],
    ...overrides,
  }
}

function businessInsights(overrides: Partial<BusinessInsightsReport> = {}): BusinessInsightsReport {
  return { insights: [], ...overrides }
}

function salesTrends(overrides: Partial<SalesTrendsReport> = {}): SalesTrendsReport {
  return {
    currency_code: 'EUR', granularity: 'day',
    points: [
      { period_start: '2026-08-01T00:00:00Z', revenue_minor_units: 40000, order_count: 3, units_sold: 10 },
      { period_start: '2026-08-02T00:00:00Z', revenue_minor_units: 60000, order_count: 5, units_sold: 14 },
    ],
    by_weekday: [
      { weekday: 1, revenue_minor_units: 70000, order_count: 6 },
      { weekday: 3, revenue_minor_units: 30000, order_count: 2 },
    ],
    ...overrides,
  }
}

function productLifecycle(overrides: Partial<ProductLifecycleReport> = {}): ProductLifecycleReport {
  return {
    currency_code: 'EUR',
    rows: [
      { sku_id: 'sku-growing', sku_code: 'GROW-1', product_name: 'Growing Widget', current_quantity_sold: 10, previous_quantity_sold: 4, current_revenue_minor_units: 1000, previous_revenue_minor_units: 400, stage: 'growing' },
      { sku_id: 'sku-dormant', sku_code: 'DORM-1', product_name: 'Dormant Widget', current_quantity_sold: 0, previous_quantity_sold: 6, current_revenue_minor_units: 0, previous_revenue_minor_units: 600, stage: 'dormant' },
    ],
    ...overrides,
  }
}

function customerCohorts(overrides: Partial<CustomerCohortsReport> = {}): CustomerCohortsReport {
  return {
    currency_code: 'EUR',
    cohorts: [
      {
        cohort_month: '2026-06-01T00:00:00Z', cohort_size: 4,
        retention: [
          { months_since_cohort: 0, retained_customers: 4, revenue_minor_units: 4000, retention_rate_basis_points: 10000 },
          { months_since_cohort: 1, retained_customers: 1, revenue_minor_units: 500, retention_rate_basis_points: 2500 },
        ],
      },
    ],
    ...overrides,
  }
}

function customerRFM(overrides: Partial<CustomerRFMReport> = {}): CustomerRFMReport {
  return {
    currency_code: 'EUR',
    rows: [
      {
        customer_id: 'cust-champ', customer_name: 'Champion Customer', order_count: 5, revenue_minor_units: 50000,
        average_order_value_minor_units: 10000, first_order_at: '2026-06-01T00:00:00Z', last_order_at: '2026-08-01T00:00:00Z',
        recency_score: 5, frequency_score: 5, monetary_score: 5, segment: 'champions',
      },
      {
        customer_id: 'cust-lost', customer_name: 'Lost Customer', order_count: 1, revenue_minor_units: 1000,
        average_order_value_minor_units: 1000, first_order_at: '2026-01-01T00:00:00Z', last_order_at: '2026-01-05T00:00:00Z',
        recency_score: 1, frequency_score: 1, monetary_score: 1, segment: 'lost',
      },
    ],
    ...overrides,
  }
}

function frequentlyBoughtTogether(overrides: Partial<FrequentlyBoughtTogetherReport> = {}): FrequentlyBoughtTogetherReport {
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
  vi.mocked(reportsApi.getCategoryPerformanceReport).mockResolvedValue(categoryPerformance())
  vi.mocked(reportsApi.getSlowMovingProductsReport).mockResolvedValue(slowMovingProducts())
  vi.mocked(reportsApi.getProfitabilityReport).mockResolvedValue(profitability())
  vi.mocked(reportsApi.getInventorySalesReport).mockResolvedValue(inventorySales())
  vi.mocked(reportsApi.getBusinessInsights).mockResolvedValue(businessInsights())
  vi.mocked(reportsApi.getSalesTrendsReport).mockResolvedValue(salesTrends())
  vi.mocked(reportsApi.getProductLifecycleReport).mockResolvedValue(productLifecycle())
  vi.mocked(reportsApi.getCustomerCohortsReport).mockResolvedValue(customerCohorts())
  vi.mocked(reportsApi.getCustomerRFMReport).mockResolvedValue(customerRFM())
  vi.mocked(reportsApi.getFrequentlyBoughtTogetherReport).mockResolvedValue(frequentlyBoughtTogether())
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

  it('shows revenue by category, including the Uncategorized bucket', async () => {
    render(<Reports />)

    expect(await screen.findByText('Beverages')).toBeInTheDocument()
    expect(screen.getByText('€400.00')).toBeInTheDocument()
    expect(screen.getByText('Uncategorized')).toBeInTheDocument()
  })

  it('shows an empty state when no category has sales in the period', async () => {
    vi.mocked(reportsApi.getCategoryPerformanceReport).mockResolvedValue(categoryPerformance({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No category sales in this period')).toBeInTheDocument()
  })

  it('shows slow-moving products with stock on hand and zero sales', async () => {
    render(<Reports />)

    expect(await screen.findByText('Slow Widget')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('shows an empty state when nothing is slow-moving', async () => {
    vi.mocked(reportsApi.getSlowMovingProductsReport).mockResolvedValue(slowMovingProducts({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No slow-moving products in this period')).toBeInTheDocument()
  })

  it('shows revenue, COGS, profit, and margin for the cost-known subset', async () => {
    render(<Reports />)

    const summary = within(await screen.findByTestId('profitability-summary'))
    expect(summary.getByText('€2,200.00')).toBeInTheDocument()
    expect(summary.getByText('€1,000.00')).toBeInTheDocument()
    expect(summary.getByText('€800.00')).toBeInTheDocument()
    expect(summary.getByText('44.4%')).toBeInTheDocument()
    expect(summary.getByText('€1,800.00 with known cost')).toBeInTheDocument()
  })

  it('shows an empty state when nothing has a known cost price for the period', async () => {
    vi.mocked(reportsApi.getProfitabilityReport).mockResolvedValue(profitability({
      summary: { revenue_minor_units: 5000, cost_known_revenue_minor_units: 0, cost_minor_units: 0, profit_minor_units: 0, margin_basis_points: 0 },
    }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No cost-known revenue in this period')).toBeInTheDocument()
  })

  it('shows stock on hand, units sold, and turnover for each SKU', async () => {
    render(<Reports />)

    expect(await screen.findByText('Turning Widget')).toBeInTheDocument()
    expect(screen.getByText('25.0%')).toBeInTheDocument()
  })

  it('shows n/a turnover for a SKU with no stock on hand', async () => {
    vi.mocked(reportsApi.getInventorySalesReport).mockResolvedValue(inventorySales({
      rows: [{ sku_id: 'sku-out', sku_code: 'OUT-1', product_name: 'Sold Out Widget', stock_on_hand: 0, quantity_sold_in_period: 6, turnover_basis_points: null }],
    }))

    render(<Reports />)

    expect(await screen.findByText('Sold Out Widget')).toBeInTheDocument()
    expect(screen.getByText('n/a')).toBeInTheDocument()
  })

  it('shows an empty state when nothing is stocked or sold', async () => {
    vi.mocked(reportsApi.getInventorySalesReport).mockResolvedValue(inventorySales({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No stocked or sold SKUs in this period')).toBeInTheDocument()
  })

  it('shows business insight banners with severity styling', async () => {
    vi.mocked(reportsApi.getBusinessInsights).mockResolvedValue(businessInsights({
      insights: [
        { code: 'revenue_declining', severity: 'warning', message: 'Revenue is down 12.5% vs the previous period.' },
        { code: 'incomplete_cost_data', severity: 'info', message: 'Only 40.0% of revenue has a known cost price.' },
      ],
    }))

    render(<Reports />)

    expect(await screen.findByText('Revenue is down 12.5% vs the previous period.')).toBeInTheDocument()
    expect(screen.getByText('Only 40.0% of revenue has a known cost price.')).toBeInTheDocument()
  })

  it('shows no insights section when nothing fired', async () => {
    render(<Reports />)

    await screen.findByText('€1,180.00')
    expect(screen.queryByTestId('business-insights')).not.toBeInTheDocument()
  })

  it('fetches sales trends with the default day granularity', async () => {
    render(<Reports />)

    await screen.findByText('€1,180.00')
    expect(reportsApi.getSalesTrendsReport).toHaveBeenCalledWith('test-token', expect.any(String), expect.any(String), 'day')
  })

  it('refetches sales trends with week granularity when switched', async () => {
    render(<Reports />)
    await screen.findByText('€1,180.00')

    await userEvent.click(screen.getByRole('button', { name: 'week' }))

    await waitFor(() => {
      expect(reportsApi.getSalesTrendsReport).toHaveBeenLastCalledWith('test-token', expect.any(String), expect.any(String), 'week')
    })
  })

  it('shows an empty state when there is no sales trend data', async () => {
    vi.mocked(reportsApi.getSalesTrendsReport).mockResolvedValue(salesTrends({ points: [] }))

    render(<Reports />)

    expect(await screen.findByText('No sales in this period')).toBeInTheDocument()
  })

  it('shows an empty state when there is no weekday breakdown', async () => {
    vi.mocked(reportsApi.getSalesTrendsReport).mockResolvedValue(salesTrends({ by_weekday: [] }))

    render(<Reports />)

    expect(await screen.findByText('No sales to break down by weekday in this period')).toBeInTheDocument()
  })

  it('places each SKU in its lifecycle stage', async () => {
    render(<Reports />)

    const growing = await screen.findByTestId('lifecycle-growing')
    expect(within(growing).getByText('Growing Widget')).toBeInTheDocument()

    const dormant = screen.getByTestId('lifecycle-dormant')
    expect(within(dormant).getByText('Dormant Widget')).toBeInTheDocument()

    expect(within(screen.getByTestId('lifecycle-stable')).getByText('None')).toBeInTheDocument()
  })

  it('shows an empty state when nothing sold in either period', async () => {
    vi.mocked(reportsApi.getProductLifecycleReport).mockResolvedValue(productLifecycle({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No sales in this or the previous period')).toBeInTheDocument()
  })

  it('shows cohort retention rates by month since acquisition', async () => {
    render(<Reports />)

    expect(await screen.findByText('Jun 2026')).toBeInTheDocument()
    expect(screen.getByText('100%')).toBeInTheDocument()
    expect(screen.getByText('25%')).toBeInTheDocument()
  })

  it('shows an empty state when no cohorts were acquired in this period', async () => {
    vi.mocked(reportsApi.getCustomerCohortsReport).mockResolvedValue(customerCohorts({ cohorts: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No new customers acquired in this period')).toBeInTheDocument()
  })

  it('places each customer in its RFM segment', async () => {
    render(<Reports />)

    const champions = await screen.findByTestId('rfm-champions')
    expect(within(champions).getByText('Champion Customer')).toBeInTheDocument()

    const lost = screen.getByTestId('rfm-lost')
    expect(within(lost).getByText('Lost Customer')).toBeInTheDocument()

    expect(within(screen.getByTestId('rfm-at_risk')).getByText('None')).toBeInTheDocument()
  })

  it('shows an empty state when no customers ordered in this period', async () => {
    vi.mocked(reportsApi.getCustomerRFMReport).mockResolvedValue(customerRFM({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No customer orders in this period')).toBeInTheDocument()
  })

  it('shows frequently bought together itemsets with support and percent', async () => {
    render(<Reports />)

    expect(await screen.findByText('Acai 1L + Granola 500g')).toBeInTheDocument()
    const row = screen.getByTestId('fbt-row')
    expect(within(row).getByText('8')).toBeInTheDocument()
    expect(within(row).getByText('40.0%')).toBeInTheDocument()
  })

  it('shows an empty state when no SKU sets co-occurred frequently in this period', async () => {
    vi.mocked(reportsApi.getFrequentlyBoughtTogetherReport).mockResolvedValue(frequentlyBoughtTogether({ rows: [] }))

    render(<Reports />)
    await screen.findByText('€1,180.00')

    expect(screen.getByText('No frequently co-occurring SKU sets in this period')).toBeInTheDocument()
  })
})
