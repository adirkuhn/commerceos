// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import OverviewPage from '../OverviewPage'
import type { SalesSummaryReport } from '../../../reports/types'
import type { BusinessInsightsReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getSalesSummaryReport: vi.fn(),
  getBusinessInsights: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

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

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getSalesSummaryReport).mockResolvedValue(salesSummary())
  vi.mocked(reportsApi.getBusinessInsights).mockResolvedValue(businessInsights())
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
})
