// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import InventorySalesPage from '../InventorySalesPage'
import { renderAt } from '../../../test/renderAt'
import type { InventorySalesReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getInventorySalesReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function inventorySales(overrides: Partial<InventorySalesReport> = {}): InventorySalesReport {
  return {
    rows: [{ sku_id: 'sku-1', sku_code: 'ACAI-1L', product_name: 'Acai 1L', stock_on_hand: 40, quantity_sold_in_period: 10, turnover_basis_points: 2500 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getInventorySalesReport).mockResolvedValue(inventorySales())
})

describe('InventorySalesPage', () => {
  it('shows stock vs. sold with turnover', async () => {
    renderAt(<InventorySalesPage />, '/reports')

    expect(await screen.findByText('Acai 1L')).toBeInTheDocument()
    expect(screen.getByText('25.0%')).toBeInTheDocument()
  })

  it('shows an empty state when there are no stocked or sold SKUs', async () => {
    vi.mocked(reportsApi.getInventorySalesReport).mockResolvedValue(inventorySales({ rows: [] }))

    renderAt(<InventorySalesPage />, '/reports')

    expect(await screen.findByText('No stocked or sold SKUs in this period')).toBeInTheDocument()
  })
})
