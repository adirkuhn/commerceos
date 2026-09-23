// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import SalesTrendsPage from '../SalesTrendsPage'
import { renderAt } from '../../../test/renderAt'
import type { SalesTrendsReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getSalesTrendsReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

function salesTrends(overrides: Partial<SalesTrendsReport> = {}): SalesTrendsReport {
  return {
    currency_code: 'EUR',
    granularity: 'day',
    points: [{ period_start: '2026-08-01T00:00:00Z', revenue_minor_units: 10000, order_count: 5, units_sold: 8 }],
    by_weekday: [{ weekday: 1, revenue_minor_units: 5000, order_count: 2 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.mocked(reportsApi.getSalesTrendsReport).mockResolvedValue(salesTrends())
})

describe('SalesTrendsPage', () => {
  it('fetches with the default day granularity', async () => {
    renderAt(<SalesTrendsPage />, '/reports')

    await screen.findByText('Sales trend')
    expect(reportsApi.getSalesTrendsReport).toHaveBeenCalledWith('test-token', expect.any(String), expect.any(String), 'day')
  })

  it('refetches with the chosen granularity when switched', async () => {
    const user = userEvent.setup()
    renderAt(<SalesTrendsPage />, '/reports')
    await screen.findByText('Sales trend')

    await user.click(screen.getByRole('button', { name: 'week' }))

    expect(reportsApi.getSalesTrendsReport).toHaveBeenLastCalledWith('test-token', expect.any(String), expect.any(String), 'week')
  })

  it('shows empty states when there is no data', async () => {
    vi.mocked(reportsApi.getSalesTrendsReport).mockResolvedValue(salesTrends({ points: [], by_weekday: [] }))

    renderAt(<SalesTrendsPage />, '/reports')

    expect(await screen.findByText('No sales in this period')).toBeInTheDocument()
    expect(screen.getByText('No sales to break down by weekday in this period')).toBeInTheDocument()
  })
})

describe('SalesTrendsPage URL state', () => {
  it('granularity and period are read from the URL', async () => {
    renderAt(<SalesTrendsPage />, '/reports', '/reports?granularity=month&period=12M')
    await waitFor(() => expect(reportsApi.getSalesTrendsReport).toHaveBeenCalledWith('test-token', expect.any(String), expect.any(String), 'month'))
    const [, from, to] = vi.mocked(reportsApi.getSalesTrendsReport).mock.calls[0]
    const days = (new Date(to).getTime() - new Date(from).getTime()) / 86_400_000
    expect(Math.round(days)).toBe(365)
  })
})
