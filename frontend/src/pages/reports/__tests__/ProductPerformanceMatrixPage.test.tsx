// @vitest-environment jsdom
import { screen, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ProductPerformanceMatrixPage from '../ProductPerformanceMatrixPage'
import { renderAt } from '../../../test/renderAt'
import type { ProductPerformanceMatrix } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getProductPerformanceMatrix: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function matrix(overrides: Partial<ProductPerformanceMatrix> = {}): ProductPerformanceMatrix {
  return {
    currency_code: 'EUR',
    median_revenue_minor_units: 10000,
    median_margin_basis_points: 3000,
    rows: [
      {
        sku_id: 'sku-1', sku_code: 'ACAI-1L', product_name: 'Acai 1L', quadrant: 'winners',
        quantity_sold: 30, revenue_minor_units: 50000, order_count: 20,
        first_sale_at: '2026-08-01T00:00:00Z', last_sale_at: '2026-08-15T00:00:00Z',
        cost_price_minor_units: 300, profit_minor_units: 41000, margin_basis_points: 4000,
      },
    ],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getProductPerformanceMatrix).mockResolvedValue(matrix())
})

describe('ProductPerformanceMatrixPage', () => {
  it('places each SKU in its quadrant', async () => {
    renderAt(<ProductPerformanceMatrixPage />, '/reports')

    const winners = await screen.findByTestId('quadrant-winners')
    expect(within(winners).getByText('Acai 1L')).toBeInTheDocument()
    expect(within(screen.getByTestId('quadrant-problems')).getByText('None')).toBeInTheDocument()
  })

  it('shows an empty state when no SKUs have a known margin', async () => {
    vi.mocked(reportsApi.getProductPerformanceMatrix).mockResolvedValue(matrix({ rows: [] }))

    renderAt(<ProductPerformanceMatrixPage />, '/reports')

    expect(await screen.findByText('No SKUs with a known margin in this period')).toBeInTheDocument()
  })
})
