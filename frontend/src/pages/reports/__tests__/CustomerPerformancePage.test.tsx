// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import CustomerPerformancePage from '../CustomerPerformancePage'
import { renderAt } from '../../../test/renderAt'
import type { CustomerPerformanceReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getCustomerPerformanceReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function customerPerformance(overrides: Partial<CustomerPerformanceReport> = {}): CustomerPerformanceReport {
  return {
    currency_code: 'EUR',
    rows: [{
      customer_id: 'cust-1', customer_name: 'Jane Doe', order_count: 5,
      revenue_minor_units: 50000, average_order_value_minor_units: 10000,
      first_order_at: '2026-06-01T00:00:00Z', last_order_at: '2026-08-01T00:00:00Z',
    }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getCustomerPerformanceReport).mockResolvedValue(customerPerformance())
})

describe('CustomerPerformancePage', () => {
  it('shows top customers by revenue', async () => {
    renderAt(<CustomerPerformancePage />, '/reports')

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('€500.00')).toBeInTheDocument()
  })

  it('shows an empty state when there are no customer orders', async () => {
    vi.mocked(reportsApi.getCustomerPerformanceReport).mockResolvedValue(customerPerformance({ rows: [] }))

    renderAt(<CustomerPerformancePage />, '/reports')

    expect(await screen.findByText('No customer orders in this period')).toBeInTheDocument()
  })
})
