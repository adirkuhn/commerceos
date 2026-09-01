// @vitest-environment jsdom
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import OverviewPage from '../OverviewPage'
import type { SalesSummaryReport, BusinessInsightsReport, ProfitabilityReport, ProfitabilityTrendReport, RevenueByChannelReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getSalesSummaryReport: vi.fn(),
  getBusinessInsights: vi.fn(),
  getProfitabilityReport: vi.fn(),
  getProfitabilityTrendReport: vi.fn(),
  getRevenueByChannelReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

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

function businessInsights(overrides: Partial<BusinessInsightsReport> = {}): BusinessInsightsReport {
  return { insights: [], ...overrides }
}

function profitability(overrides: Partial<ProfitabilityReport> = {}): ProfitabilityReport {
  return {
    currency_code: 'EUR',
    summary: {
      revenue_minor_units: 100000, cost_known_revenue_minor_units: 80000,
      cost_minor_units: 40000, profit_minor_units: 40000, margin_basis_points: 5000,
    },
    ...overrides,
  }
}

function profitabilityTrend(overrides: Partial<ProfitabilityTrendReport> = {}): ProfitabilityTrendReport {
  return {
    currency_code: 'EUR', granularity: 'day',
    points: [{ period_start: '2026-08-01T00:00:00Z', revenue_minor_units: 1000, cost_known_revenue_minor_units: 800, cost_minor_units: 400, profit_minor_units: 400, margin_basis_points: 5000 }],
    ...overrides,
  }
}

function revenueByChannel(overrides: Partial<RevenueByChannelReport> = {}): RevenueByChannelReport {
  return {
    currency_code: 'EUR', granularity: 'day',
    channels: [{ source: 'woocommerce', total_minor_units: 31500 }],
    points: [{ period_start: '2026-08-01T00:00:00Z', source: 'woocommerce', revenue_minor_units: 31500 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.mocked(reportsApi.getSalesSummaryReport).mockResolvedValue(salesSummary())
  vi.mocked(reportsApi.getBusinessInsights).mockResolvedValue(businessInsights())
  vi.mocked(reportsApi.getProfitabilityReport).mockResolvedValue(profitability())
  vi.mocked(reportsApi.getProfitabilityTrendReport).mockResolvedValue(profitabilityTrend())
  vi.mocked(reportsApi.getRevenueByChannelReport).mockResolvedValue(revenueByChannel())
})

describe('OverviewPage', () => {
  it('shows revenue and order KPIs with period-over-period change', async () => {
    render(<OverviewPage />)

    expect(await screen.findByText('€1,180.00')).toBeInTheDocument()
    expect(screen.getByText('+18.4%')).toBeInTheDocument()
    expect(screen.getByText('112')).toBeInTheDocument()
  })

  it('shows business insight banners when the report has any', async () => {
    vi.mocked(reportsApi.getBusinessInsights).mockResolvedValue(businessInsights({
      insights: [{ code: 'revenue_declining', severity: 'warning', message: 'Revenue is down' }],
    }))

    render(<OverviewPage />)

    expect(await screen.findByTestId('business-insights')).toBeInTheDocument()
    expect(screen.getByText('Revenue is down')).toBeInTheDocument()
  })

  it('shows the profitability summary by default', async () => {
    render(<OverviewPage />)

    const summary = await screen.findByTestId('profitability-summary')
    expect(summary).toBeInTheDocument()
    expect(screen.getByText('50.0%')).toBeInTheDocument()
  })

  it('shows an empty state when there is no cost-known revenue', async () => {
    vi.mocked(reportsApi.getProfitabilityReport).mockResolvedValue(profitability({
      summary: { revenue_minor_units: 0, cost_known_revenue_minor_units: 0, cost_minor_units: 0, profit_minor_units: 0, margin_basis_points: 0 },
    }))

    render(<OverviewPage />)

    expect(await screen.findByText('No cost-known revenue in this period')).toBeInTheDocument()
  })

  it('shows revenue per channel as a bar chart and table by default', async () => {
    render(<OverviewPage />)

    expect(await screen.findByText('woocommerce')).toBeInTheDocument()
    expect(screen.getByText('€315.00')).toBeInTheDocument()
  })

  it('shows an empty state when there is no channel revenue', async () => {
    vi.mocked(reportsApi.getRevenueByChannelReport).mockResolvedValue(revenueByChannel({ channels: [] }))

    render(<OverviewPage />)

    expect(await screen.findAllByText('No revenue in this period')).not.toHaveLength(0)
  })

  it('switches Profitability and Revenue by Channel to a daily view', async () => {
    render(<OverviewPage />)
    await screen.findByTestId('profitability-summary')

    fireEvent.click(screen.getByText('Daily'))

    expect(await screen.findByText('Daily revenue per sales channel')).toBeInTheDocument()
    expect(screen.queryByTestId('profitability-summary')).not.toBeInTheDocument()
    expect(reportsApi.getProfitabilityTrendReport).toHaveBeenCalledWith('test-token', expect.any(String), expect.any(String), 'day')
    expect(reportsApi.getRevenueByChannelReport).toHaveBeenCalledWith('test-token', expect.any(String), expect.any(String), 'day')
  })

  it('reveals custom date inputs and refetches with the chosen range', async () => {
    render(<OverviewPage />)
    await screen.findByTestId('profitability-summary')
    vi.mocked(reportsApi.getSalesSummaryReport).mockClear()

    fireEvent.click(screen.getByText('Custom'))
    const inputs = document.querySelectorAll('input[type="date"]')
    expect(inputs).toHaveLength(2)

    fireEvent.change(inputs[0], { target: { value: '2026-01-01' } })
    fireEvent.change(inputs[1], { target: { value: '2026-01-31' } })

    await screen.findByTestId('profitability-summary')
    expect(reportsApi.getSalesSummaryReport).toHaveBeenCalledWith('test-token', '2026-01-01T00:00:00.000Z', '2026-02-01T00:00:00.000Z')
  })
})
