// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import PurchasePlanner from '../PurchasePlanner'
import { renderAt } from '../../test/renderAt'
import type { ReorderRecommendations } from '../../purchaseplanning/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))
vi.mock('../../purchaseplanning/api', () => ({ getReorderRecommendations: vi.fn() }))
vi.mock('../../suppliers/api', () => ({
  listSuppliers: vi.fn(() => Promise.resolve([{ id: 'sup-1', name: 'Bauducco NL', status: 'active' }])),
}))

import * as api from '../../purchaseplanning/api'

const plan: ReorderRecommendations = {
  as_of: '2026-10-06', coverage_days: 30, safety_days: 7,
  suppliers: [{
    supplier_id: 'sup-1', supplier_name: 'Bauducco NL', lead_time_days: 21, lead_time_assumed: false,
    estimated_cost_minor_units: 77400, currency_code: 'EUR', earliest_order_by_date: '2026-10-09',
    lines: [{
      sku_id: 'sku-1', sku_code: 'PAN-1', product_name: 'Panettone', variant_name: 'Default', is_weight_pool: false,
      on_hand: 12, on_order: { sent: 0, draft: 0 },
      forecast: { units: 968, method: 'seasonal', growth_factor: 1.18, days: 51, recent_units: 118, last_year_units: 820 },
      daily_demand: 19, safety_stock: 133, needed_units: 1031, purchase_quantity: 172, units_per_purchased_item: 6,
      order_units: 1032, unit_cost_minor_units: 450, estimated_cost_minor_units: 77400, currency_code: 'EUR',
      order_by_date: '2026-10-09',
    }],
  }],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getReorderRecommendations).mockResolvedValue(plan)
})

describe('PurchasePlanner', () => {
  it('loads the default window and lists recommendations per supplier', async () => {
    renderAt(<PurchasePlanner />, '/purchase-planner')

    expect(await screen.findByRole('heading', { name: 'Bauducco NL' })).toBeInTheDocument()
    expect(screen.getByText('Panettone')).toBeInTheDocument()
    expect(api.getReorderRecommendations).toHaveBeenCalledWith('test-token', { coverage_days: 30, safety_days: 7, supplier_id: null })
  })

  it('reads the window from the URL and writes changes back to it', async () => {
    const { router } = renderAt(<PurchasePlanner />, '/purchase-planner', '/purchase-planner?coverage=60&safety=0')
    await screen.findByText('Panettone')
    expect(api.getReorderRecommendations).toHaveBeenLastCalledWith('test-token', { coverage_days: 60, safety_days: 0, supplier_id: null })

    await userEvent.selectOptions(screen.getByLabelText('Cover'), '90')

    await waitFor(() => expect(api.getReorderRecommendations).toHaveBeenLastCalledWith('test-token', { coverage_days: 90, safety_days: 0, supplier_id: null }))
    expect(router.state.location.search).toContain('coverage=90')
  })

  it('filters by supplier', async () => {
    const { router } = renderAt(<PurchasePlanner />, '/purchase-planner')
    await screen.findByText('Panettone')
    await screen.findByRole('option', { name: 'Bauducco NL' })

    await userEvent.selectOptions(screen.getByLabelText('Supplier'), 'sup-1')

    await waitFor(() => expect(api.getReorderRecommendations).toHaveBeenLastCalledWith('test-token', { coverage_days: 30, safety_days: 7, supplier_id: 'sup-1' }))
    expect(router.state.location.search).toContain('supplier=sup-1')
  })

  it('shows an empty state', async () => {
    vi.mocked(api.getReorderRecommendations).mockResolvedValue({ ...plan, suppliers: [] })
    renderAt(<PurchasePlanner />, '/purchase-planner')

    expect(await screen.findByText('Nothing to reorder for this period.')).toBeInTheDocument()
  })

  it('shows API errors', async () => {
    vi.mocked(api.getReorderRecommendations).mockRejectedValue(new Error('coverage_days must be 1-365'))
    renderAt(<PurchasePlanner />, '/purchase-planner')

    expect(await screen.findByText('coverage_days must be 1-365')).toBeInTheDocument()
  })
})
