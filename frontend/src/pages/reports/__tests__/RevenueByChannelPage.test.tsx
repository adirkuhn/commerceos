// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import RevenueByChannelPage from '../RevenueByChannelPage'
import type { DashboardStatsReport } from '../../../dashboard/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../dashboard/api', () => ({
  getDashboardStats: vi.fn(),
}))

import * as dashboardApi from '../../../dashboard/api'

// jsdom doesn't implement ResizeObserver, which recharts' ResponsiveContainer requires to mount.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
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
  vi.mocked(dashboardApi.getDashboardStats).mockResolvedValue(dashboardStats())
})

describe('RevenueByChannelPage', () => {
  it('shows revenue per channel for this month', async () => {
    render(<RevenueByChannelPage />)

    expect(await screen.findByText('woocommerce')).toBeInTheDocument()
    expect(screen.getByText('€315.00')).toBeInTheDocument()
  })

  it('shows an empty state when there is no revenue this month', async () => {
    vi.mocked(dashboardApi.getDashboardStats).mockResolvedValue(dashboardStats({ sales_by_channel_this_month: [] }))

    render(<RevenueByChannelPage />)

    expect(await screen.findByText('No revenue this month')).toBeInTheDocument()
  })
})
