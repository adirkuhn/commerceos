// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ProductLifecyclePage from '../ProductLifecyclePage'
import type { ProductLifecycleReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getProductLifecycleReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

function productLifecycle(overrides: Partial<ProductLifecycleReport> = {}): ProductLifecycleReport {
  return {
    currency_code: 'EUR',
    rows: [{
      sku_id: 'sku-1', sku_code: 'ACAI-1L', product_name: 'Acai 1L', stage: 'growing',
      current_quantity_sold: 50, previous_quantity_sold: 20,
      current_revenue_minor_units: 25000, previous_revenue_minor_units: 10000,
    }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reportsApi.getProductLifecycleReport).mockResolvedValue(productLifecycle())
})

describe('ProductLifecyclePage', () => {
  it('places each SKU in its lifecycle stage', async () => {
    render(<ProductLifecyclePage />)

    const growing = await screen.findByTestId('lifecycle-growing')
    expect(within(growing).getByText('Acai 1L')).toBeInTheDocument()
    expect(within(screen.getByTestId('lifecycle-dormant')).getByText('None')).toBeInTheDocument()
  })

  it('shows an empty state when there are no sales in either period', async () => {
    vi.mocked(reportsApi.getProductLifecycleReport).mockResolvedValue(productLifecycle({ rows: [] }))

    render(<ProductLifecyclePage />)

    expect(await screen.findByText('No sales in this or the previous period')).toBeInTheDocument()
  })
})
