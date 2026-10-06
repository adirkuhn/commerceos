// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import PurchasePlanner from '../PurchasePlanner'
import { renderAt } from '../../test/renderAt'
import type { ReorderRecommendations, SeasonEvent, SeasonPlan } from '../../purchaseplanning/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))
vi.mock('../../purchaseplanning/api', () => ({ getReorderRecommendations: vi.fn(), listSeasonEvents: vi.fn(), getSeasonPlan: vi.fn(), createPurchaseOrderFromPlan: vi.fn() }))
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

const events: SeasonEvent[] = [
  { key: 'black_friday', year: 2026, date: '2026-11-27', window: { start: '2026-11-20', end: '2026-12-01' } },
  { key: 'christmas', year: 2026, date: '2026-12-25', window: { start: '2026-11-13', end: '2026-12-27' } },
]

const seasonPlan: SeasonPlan = {
  as_of: '2026-10-06', event: 'christmas', year: 2026, safety_days: 7,
  window: { start: '2026-11-13', end: '2026-12-27' }, last_year_window: { start: '2025-11-13', end: '2025-12-27' },
  suppliers: plan.suppliers.map(s => ({ ...s, lines: s.lines.map(l => ({ ...l, seasonal_lift: 18.6 })) })),
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getReorderRecommendations).mockResolvedValue(plan)
  vi.mocked(api.listSeasonEvents).mockResolvedValue(events)
  vi.mocked(api.getSeasonPlan).mockResolvedValue(seasonPlan)
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

  it('turns a supplier\'s recommendations into a draft PO and opens it', async () => {
    vi.mocked(api.createPurchaseOrderFromPlan).mockResolvedValue({ purchase_order_id: 'po-9' })
    const { router } = renderAt(<PurchasePlanner />, '/purchase-planner')
    await screen.findByText('Panettone')

    await userEvent.click(screen.getByRole('button', { name: 'Create draft PO' }))

    expect(api.createPurchaseOrderFromPlan).toHaveBeenCalledWith('test-token', { supplier_id: 'sup-1', lines: [{ sku_id: 'sku-1', quantity: 172 }] })
    await waitFor(() => expect(router.state.location.pathname).toBe('/procurement/po-9'))
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

describe('PurchasePlanner season tab', () => {
  it('plans the next upcoming holiday by default', async () => {
    renderAt(<PurchasePlanner />, '/purchase-planner', '/purchase-planner?tab=season')

    expect(await screen.findByText('Panettone')).toBeInTheDocument()
    expect(api.getSeasonPlan).toHaveBeenLastCalledWith('test-token', {
      event: 'black_friday', from: null, to: null, safety_days: 7, supplier_id: null, all_products: false,
    })
    expect(screen.getByText('18.6× usual sales')).toBeInTheDocument()
  })

  it('shows the season and last year\'s matching window', async () => {
    renderAt(<PurchasePlanner />, '/purchase-planner', '/purchase-planner?tab=season&season=christmas')

    expect(await screen.findByText('Christmas 2026')).toBeInTheDocument()
    expect(screen.getByText(/13\/11\/2026 – 26\/12\/2026/)).toBeInTheDocument()
    expect(screen.getByText('· compared with 13/11/2025 – 26/12/2025', { exact: false })).toBeInTheDocument()
  })

  it('switches tabs through the URL', async () => {
    const { router } = renderAt(<PurchasePlanner />, '/purchase-planner')
    await screen.findByText('Panettone')

    await userEvent.click(screen.getByRole('tab', { name: 'Season plan' }))

    expect(router.state.location.search).toContain('tab=season')
    await waitFor(() => expect(api.getSeasonPlan).toHaveBeenCalled())
  })

  it('plans a custom date range, sending the end as exclusive', async () => {
    const { router } = renderAt(<PurchasePlanner />, '/purchase-planner', '/purchase-planner?tab=season&season=custom')
    await screen.findByLabelText('From')
    expect(api.getSeasonPlan).not.toHaveBeenCalled()

    router.navigate('/purchase-planner?tab=season&season=custom&from=2026-11-01&to=2026-11-10&all=1')

    await waitFor(() => expect(api.getSeasonPlan).toHaveBeenLastCalledWith('test-token', {
      event: null, from: '2026-11-01', to: '2026-11-11', safety_days: 7, supplier_id: null, all_products: true,
    }))
    expect(screen.getByLabelText('Include all products')).toBeChecked()
  })

  it('shows season errors', async () => {
    vi.mocked(api.getSeasonPlan).mockRejectedValue(new Error('this season is already over'))
    renderAt(<PurchasePlanner />, '/purchase-planner', '/purchase-planner?tab=season&season=easter')

    expect(await screen.findByText('this season is already over')).toBeInTheDocument()
  })
})
