// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import StockoutRiskPage from '../StockoutRiskPage'
import type { StockoutRiskReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getStockoutRiskReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function stockoutRisk(overrides: Partial<StockoutRiskReport> = {}): StockoutRiskReport {
  return {
    rows: [
      {
        sku_id: 'sku-a', sku_code: 'RISKY-1', product_name: 'Risky Widget',
        stock_on_hand: 5, quantity_sold_in_period: 30, daily_velocity: 1,
        days_of_stock_remaining: 5, at_risk: true, suggested_reorder_quantity: 9,
      },
      {
        sku_id: 'sku-b', sku_code: 'SAFE-1', product_name: 'Safe Widget',
        stock_on_hand: 1000, quantity_sold_in_period: 10, daily_velocity: 0.33,
        days_of_stock_remaining: 3000, at_risk: false, suggested_reorder_quantity: 0,
      },
    ],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getStockoutRiskReport).mockResolvedValue(stockoutRisk())
})

describe('StockoutRiskPage', () => {
  it('shows only at-risk SKUs, ranked soonest-first', async () => {
    render(<StockoutRiskPage />)

    const row = await screen.findByTestId('stockout-risk-row')
    expect(within(row).getByText('Risky Widget')).toBeInTheDocument()
    expect(within(row).getByText('5.0')).toBeInTheDocument()
    expect(within(row).getByText('9')).toBeInTheDocument()
    expect(screen.getAllByTestId('stockout-risk-row')).toHaveLength(1)
  })

  it('shows an empty state when no SKUs are at risk', async () => {
    vi.mocked(reportsApi.getStockoutRiskReport).mockResolvedValue(stockoutRisk({
      rows: [{
        sku_id: 'sku-b', sku_code: 'SAFE-1', product_name: 'Safe Widget',
        stock_on_hand: 1000, quantity_sold_in_period: 10, daily_velocity: 0.33,
        days_of_stock_remaining: 3000, at_risk: false, suggested_reorder_quantity: 0,
      }],
    }))

    render(<StockoutRiskPage />)

    expect(await screen.findByText('No SKUs at risk of stocking out')).toBeInTheDocument()
  })
})
