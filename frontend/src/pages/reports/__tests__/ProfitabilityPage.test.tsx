// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ProfitabilityPage from '../ProfitabilityPage'
import type { ProfitabilityReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getProfitabilityReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

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

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getProfitabilityReport).mockResolvedValue(profitability())
})

describe('ProfitabilityPage', () => {
  it('shows revenue, COGS, profit, and margin for the period', async () => {
    render(<ProfitabilityPage />)

    const summary = await screen.findByTestId('profitability-summary')
    expect(summary).toBeInTheDocument()
    expect(screen.getByText('50.0%')).toBeInTheDocument()
  })

  it('shows an empty state when there is no cost-known revenue', async () => {
    vi.mocked(reportsApi.getProfitabilityReport).mockResolvedValue(profitability({
      summary: { revenue_minor_units: 0, cost_known_revenue_minor_units: 0, cost_minor_units: 0, profit_minor_units: 0, margin_basis_points: 0 },
    }))

    render(<ProfitabilityPage />)

    expect(await screen.findByText('No cost-known revenue in this period')).toBeInTheDocument()
  })
})
