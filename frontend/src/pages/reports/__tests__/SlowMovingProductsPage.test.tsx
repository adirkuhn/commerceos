// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import SlowMovingProductsPage from '../SlowMovingProductsPage'
import { renderAt } from '../../../test/renderAt'
import type { SlowMovingProductsReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getSlowMovingProductsReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function slowMovingProducts(overrides: Partial<SlowMovingProductsReport> = {}): SlowMovingProductsReport {
  return {
    rows: [{ sku_id: 'sku-1', sku_code: 'ACAI-1L', product_name: 'Acai 1L', stock_on_hand: 40, quantity_sold_in_period: 0 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getSlowMovingProductsReport).mockResolvedValue(slowMovingProducts())
})

describe('SlowMovingProductsPage', () => {
  it('shows SKUs with stock but zero sales', async () => {
    renderAt(<SlowMovingProductsPage />, '/reports')

    expect(await screen.findByText('Acai 1L')).toBeInTheDocument()
    expect(screen.getByText('40')).toBeInTheDocument()
  })

  it('shows an empty state when there are no slow-moving products', async () => {
    vi.mocked(reportsApi.getSlowMovingProductsReport).mockResolvedValue(slowMovingProducts({ rows: [] }))

    renderAt(<SlowMovingProductsPage />, '/reports')

    expect(await screen.findByText('No slow-moving products in this period')).toBeInTheDocument()
  })
})
