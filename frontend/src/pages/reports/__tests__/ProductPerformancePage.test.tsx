// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ProductPerformancePage from '../ProductPerformancePage'
import { renderAt } from '../../../test/renderAt'
import type { ProductPerformanceReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getProductPerformanceReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

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

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getProductPerformanceReport).mockResolvedValue(productPerformance())
})

describe('ProductPerformancePage', () => {
  it('shows top products by revenue with margin', async () => {
    renderAt(<ProductPerformancePage />, '/reports')

    expect(await screen.findByText('Acai 1L')).toBeInTheDocument()
    expect(screen.getByText('€5,850.00')).toBeInTheDocument()
    expect(screen.getByText('88.0%')).toBeInTheDocument()
  })

  it('shows an empty state when there are no sales', async () => {
    vi.mocked(reportsApi.getProductPerformanceReport).mockResolvedValue(productPerformance({ rows: [] }))

    renderAt(<ProductPerformancePage />, '/reports')

    expect(await screen.findByText('No sales in this period')).toBeInTheDocument()
  })
})

describe('ProductPerformancePage URL state', () => {
  it('reads the period from ?period=', async () => {
    renderAt(<ProductPerformancePage />, '/reports', '/reports?period=30D')
    await waitFor(() => expect(reportsApi.getProductPerformanceReport).toHaveBeenCalled())
    const [, from, to] = vi.mocked(reportsApi.getProductPerformanceReport).mock.calls[0]
    const days = (new Date(to as string).getTime() - new Date(from as string).getTime()) / 86_400_000
    expect(Math.round(days)).toBe(30)
  })
})
