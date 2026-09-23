// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import CustomerCohortsPage from '../CustomerCohortsPage'
import { renderAt } from '../../../test/renderAt'
import type { CustomerCohortsReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getCustomerCohortsReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function customerCohorts(overrides: Partial<CustomerCohortsReport> = {}): CustomerCohortsReport {
  return {
    currency_code: 'EUR',
    cohorts: [
      {
        cohort_month: '2026-06-01T00:00:00Z', cohort_size: 4,
        retention: [
          { months_since_cohort: 0, retained_customers: 4, revenue_minor_units: 4000, retention_rate_basis_points: 10000 },
          { months_since_cohort: 1, retained_customers: 1, revenue_minor_units: 500, retention_rate_basis_points: 2500 },
        ],
      },
    ],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getCustomerCohortsReport).mockResolvedValue(customerCohorts())
})

describe('CustomerCohortsPage', () => {
  it('shows cohort retention rates by month since acquisition', async () => {
    renderAt(<CustomerCohortsPage />, '/reports')

    expect(await screen.findByText('Jun 2026')).toBeInTheDocument()
    expect(screen.getByText('100%')).toBeInTheDocument()
    expect(screen.getByText('25%')).toBeInTheDocument()
  })

  it('shows an empty state when no cohorts were acquired in this period', async () => {
    vi.mocked(reportsApi.getCustomerCohortsReport).mockResolvedValue(customerCohorts({ cohorts: [] }))

    renderAt(<CustomerCohortsPage />, '/reports')

    expect(await screen.findByText('No new customers acquired in this period')).toBeInTheDocument()
  })
})
