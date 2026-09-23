// @vitest-environment jsdom
import { screen, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import CustomerRFMPage from '../CustomerRFMPage'
import { renderAt } from '../../../test/renderAt'
import type { CustomerRFMReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getCustomerRFMReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function customerRFM(overrides: Partial<CustomerRFMReport> = {}): CustomerRFMReport {
  return {
    currency_code: 'EUR',
    rows: [
      {
        customer_id: 'cust-champ', customer_name: 'Champion Customer', order_count: 5, revenue_minor_units: 50000,
        average_order_value_minor_units: 10000, first_order_at: '2026-06-01T00:00:00Z', last_order_at: '2026-08-01T00:00:00Z',
        recency_score: 5, frequency_score: 5, monetary_score: 5, segment: 'champions',
      },
    ],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getCustomerRFMReport).mockResolvedValue(customerRFM())
})

describe('CustomerRFMPage', () => {
  it('places each customer in its RFM segment', async () => {
    renderAt(<CustomerRFMPage />, '/reports')

    const champions = await screen.findByTestId('rfm-champions')
    expect(within(champions).getByText('Champion Customer')).toBeInTheDocument()
    expect(within(screen.getByTestId('rfm-lost')).getByText('None')).toBeInTheDocument()
  })

  it('shows an empty state when no customers ordered in this period', async () => {
    vi.mocked(reportsApi.getCustomerRFMReport).mockResolvedValue(customerRFM({ rows: [] }))

    renderAt(<CustomerRFMPage />, '/reports')

    expect(await screen.findByText('No customer orders in this period')).toBeInTheDocument()
  })
})
