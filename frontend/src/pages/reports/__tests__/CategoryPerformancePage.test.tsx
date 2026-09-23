// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import CategoryPerformancePage from '../CategoryPerformancePage'
import { renderAt } from '../../../test/renderAt'
import type { CategoryPerformanceReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getCategoryPerformanceReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function categoryPerformance(overrides: Partial<CategoryPerformanceReport> = {}): CategoryPerformanceReport {
  return {
    currency_code: 'EUR',
    rows: [{ category_id: 'cat-1', category_name: 'Beverages', quantity_sold: 100, order_count: 40, revenue_minor_units: 200000 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getCategoryPerformanceReport).mockResolvedValue(categoryPerformance())
})

describe('CategoryPerformancePage', () => {
  it('shows revenue by category', async () => {
    renderAt(<CategoryPerformancePage />, '/reports')

    expect(await screen.findByText('Beverages')).toBeInTheDocument()
    expect(screen.getByText('€2,000.00')).toBeInTheDocument()
  })

  it('shows an empty state when there are no category sales', async () => {
    vi.mocked(reportsApi.getCategoryPerformanceReport).mockResolvedValue(categoryPerformance({ rows: [] }))

    renderAt(<CategoryPerformancePage />, '/reports')

    expect(await screen.findByText('No category sales in this period')).toBeInTheDocument()
  })
})
