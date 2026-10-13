// @vitest-environment jsdom
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi } from 'vitest'
import { CloseToExpiryPanel } from '../CloseToExpiryPanel'
import type { CloseToExpiryState } from '../useCloseToExpiry'
import type { CloseToExpiryBatch, CloseToExpiryList, CloseToExpiryWindow } from '../types'

vi.mock('../../../promotions/api')
vi.mock('../../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../../tax/api', () => ({
  listTaxClasses: vi.fn(async () => [
    { id: 'tc-std', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
    { id: 'tc-food', name: 'Food', is_default: false, created_at: '', updated_at: '' },
  ]),
  listTaxRates: vi.fn(async () => [
    { id: 'r1', tax_class_id: 'tc-std', country: 'NL', rate_basis_points: 2100, created_at: '', updated_at: '' },
    { id: 'r2', tax_class_id: 'tc-food', country: 'NL', rate_basis_points: 900, created_at: '', updated_at: '' },
  ]),
}))
vi.mock('../../../settings/api', () => ({
  getTenantSettings: vi.fn(async () => ({ default_tax_country: 'NL', prices_include_tax: true })),
  updateTenantSettings: vi.fn(),
}))

const w = (window_days: number, min_days_left: number, open_count: number, acknowledged_count = 0): CloseToExpiryWindow => ({
  window_days, min_days_left, from: '2026-10-28', to: '2026-11-11', open_count, acknowledged_count,
})

const batch: CloseToExpiryBatch = {
  batch_id: 'b-1', lot_number: 'L1', expiry_date: '2026-10-30', days_left: 18, quantity: 7,
  warehouse_id: 'wh-1', warehouse_name: 'Amsterdam', sku_id: 'sku-1', sku_code: 'PQ-1',
  product_id: 'p-1', product_name: 'Pão de queijo', image_url: '', price_minor_units: 650, currency_code: 'EUR',
}

function state(overrides: Partial<CloseToExpiryState> = {}, data: Partial<CloseToExpiryList> = {}): CloseToExpiryState {
  const windows = [w(90, 31, 4), w(30, 16, 1, 2), w(15, 8, 0), w(7, 0, 3)]
  return {
    data: { today: '2026-10-12', window: windows[1], windows, items: [batch], page: 1, per_page: 50, total: 1, ...data },
    loading: false, error: null, actingId: null, actionError: '', seen: false, page: 1,
    setPage: vi.fn(), setWindow: vi.fn(), setSeen: vi.fn(), acknowledge: vi.fn(), undo: vi.fn(), refresh: vi.fn(),
    ...overrides,
  }
}

function renderPanel(s: CloseToExpiryState, onDiscard = vi.fn()) {
  render(<MemoryRouter><CloseToExpiryPanel token="tok" state={s} onDiscard={onDiscard} /></MemoryRouter>)
  return { onDiscard }
}

describe('CloseToExpiryPanel', () => {
  it('shows every window with its open count and marks the current one', () => {
    renderPanel(state())
    const tabs = screen.getByRole('group', { name: /close to expiry/i })
    expect(within(tabs).getByRole('button', { name: /90 days.*4/i })).toHaveAttribute('aria-pressed', 'false')
    expect(within(tabs).getByRole('button', { name: /30 days.*1/i })).toHaveAttribute('aria-pressed', 'true')
  })

  it('switches window and list', () => {
    const s = state()
    renderPanel(s)
    fireEvent.click(screen.getByRole('button', { name: /7 days/i }))
    expect(s.setWindow).toHaveBeenCalledWith(7)
    fireEvent.click(screen.getByRole('button', { name: /seen.*2/i }))
    expect(s.setSeen).toHaveBeenCalledWith(true)
  })

  it('lists the batch with its days-left stamp, stock and price', () => {
    renderPanel(state())
    expect(screen.getByRole('img', { name: '18 days left' })).toBeInTheDocument()
    expect(screen.getByText('Pão de queijo')).toBeInTheDocument()
    expect(screen.getByText(/Lot L1/)).toBeInTheDocument()
    expect(screen.getByText('7 in stock')).toBeInTheDocument()
    expect(screen.getByText('€6.50')).toBeInTheDocument()
  })

  it('marks as seen with a hint naming the next window', () => {
    const s = state()
    renderPanel(s)
    const mark = screen.getByRole('button', { name: /mark as seen/i })
    expect(mark).toHaveAttribute('title', expect.stringMatching(/comes back at 15 days/))
    fireEvent.click(mark)
    expect(s.acknowledge).toHaveBeenCalledWith(batch)
  })

  it('offers to show a seen batch again', () => {
    const s = state({ seen: true }, { items: [{ ...batch, acknowledged_at: '2026-10-11T09:00:00Z' }] })
    renderPanel(s)
    fireEvent.click(screen.getByRole('button', { name: /show again/i }))
    expect(s.undo).toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: /mark as seen/i })).not.toBeInTheDocument()
  })

  it('links a running clearance instead of offering a new sale', () => {
    renderPanel(state({}, { items: [{ ...batch, clearance_promotion: { id: 'promo-1', name: 'Clearance PQ' } }] }))
    expect(screen.getByRole('link', { name: /on sale: clearance pq/i })).toHaveAttribute('href', '/promotions/promo-1')
    expect(screen.queryByRole('button', { name: /put on sale/i })).not.toBeInTheDocument()
  })

  it('opens the sale dialog and discard', () => {
    const { onDiscard } = renderPanel(state())
    fireEvent.click(screen.getByRole('button', { name: /put on sale/i }))
    expect(screen.getByRole('dialog', { name: /put pão de queijo on sale/i })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /^close$/i }))
    fireEvent.click(screen.getByRole('button', { name: /discard/i }))
    expect(onDiscard).toHaveBeenCalledWith(batch)
  })

  it('says when a window has nothing left to review', () => {
    renderPanel(state({}, { items: [], total: 0 }))
    expect(screen.getByText(/nothing left to review/i)).toBeInTheDocument()
  })

  it('shows load and action errors', () => {
    renderPanel(state({ error: 'Load failed', actionError: 'Moved on' }))
    expect(screen.getByText('Load failed')).toBeInTheDocument()
    expect(screen.getByText('Moved on')).toBeInTheDocument()
  })

  it('pages through a long list', () => {
    const s = state({ page: 1 }, { total: 120, per_page: 50 })
    renderPanel(s)
    expect(screen.getByText('Page 1 of 3')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    expect(s.setPage).toHaveBeenCalledWith(2)
  })
})

describe('CloseToExpiryPanel -- put on sale profit', () => {
  // The SKU's own tax class (9% food) decides the excl. VAT numbers.
  it('opens the sale dialog with the SKU\'s VAT rate for the profit preview', async () => {
    const costed: CloseToExpiryBatch = { ...batch, price_minor_units: 450, unit_cost_minor_units: 210, unit_cost_source: 'batch', tax_class_id: 'tc-food' }
    renderPanel(state({}, { items: [costed] }))

    fireEvent.click(screen.getByRole('button', { name: /put on sale/i }))

    const panel = within(await screen.findByRole('region', { name: 'Profit' }))
    await waitFor(() => expect(panel.getByText(/€2\.03 · 49\.1% · 96\.7%/)).toBeInTheDocument()) // 4.50 / 1.09
  })
})
