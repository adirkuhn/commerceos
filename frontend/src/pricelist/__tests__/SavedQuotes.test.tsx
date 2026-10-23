// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import type { PriceListQuery, PriceListRow, QuoteInput, SavedQuote } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ listPriceList: vi.fn() }))
vi.mock('../quotesApi', () => ({
  listQuotes: vi.fn(), getQuote: vi.fn(), createQuote: vi.fn(), updateQuote: vi.fn(), deleteQuote: vi.fn(), setAutomaticCostPrice: vi.fn(),
}))
import * as api from '../api'
import * as quotesApi from '../quotesApi'
import PriceList from '../../pages/PriceList'

const P1 = '0b0e7e8e-0000-4000-8000-000000000001'
const feijao: PriceListRow = {
  product_id: P1, product_name: 'Feijão', variant_name: '', sku_id: 's-fei', sku_code: 'FEI-1', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 1000, sale_price_minor_units: null, cost_price_minor_units: 400,
  available_quantity: 40, earliest_expiry: null, tax_rate_basis_points: 0, target_markup_basis_points: 0,
}

// A tiny in-memory backend for the quote endpoints.
let store: Record<string, SavedQuote>
let nextNumber: number
function toSaved(id: string, number: number, input: QuoteInput): SavedQuote {
  const lines = input.lines.map(l => ({ ...l, line_total_minor_units: l.quantity * l.unit_price_minor_units }))
  return {
    id, number, reference: `Q-${String(number).padStart(4, '0')}`, name: input.name, currency_code: input.currency_code ?? '',
    discount_basis_points: input.discount_basis_points, line_count: lines.length,
    total_minor_units: lines.reduce((s, l) => s + l.line_total_minor_units, 0), created_at: '2026-10-23T10:00:00Z', updated_at: '2026-10-23T10:00:00Z', lines,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  store = {}
  nextNumber = 1
  vi.mocked(api.listPriceList).mockImplementation(async (_t: string, q: PriceListQuery) => {
    const rows = [feijao].filter(r => (q.productIds ? q.productIds.includes(r.product_id) : true))
    return { rows, total: rows.length, pricesIncludeTax: false }
  })
  vi.mocked(quotesApi.createQuote).mockImplementation(async (_t, input) => {
    const q = toSaved(`q-${nextNumber}`, nextNumber, input)
    nextNumber++
    store[q.id] = q
    return q
  })
  vi.mocked(quotesApi.updateQuote).mockImplementation(async (_t, id, input) => {
    store[id] = toSaved(id, store[id].number, input)
    return store[id]
  })
  vi.mocked(quotesApi.getQuote).mockImplementation(async (_t, id) => {
    if (!store[id]) throw new Error('quote not found')
    return store[id]
  })
  vi.mocked(quotesApi.listQuotes).mockImplementation(async () => ({ quotes: Object.values(store), total: Object.keys(store).length }))
  vi.mocked(quotesApi.deleteQuote).mockImplementation(async (_t, id) => { delete store[id] })
})

let search = ''
function LocationProbe() {
  search = decodeURIComponent(useLocation().search)
  return null
}

function renderPage(url: string) {
  return render(<MemoryRouter initialEntries={[url]}><PriceList /><LocationProbe /></MemoryRouter>)
}

describe('saving a quote', () => {
  it('needs a name, then saves the lines at the shown prices and gets a number', async () => {
    renderPage(`/price-list?tab=quote&discount=20&lines=${P1}.s-fei.10`)
    await screen.findByTestId('quote-line-s-fei')
    const save = screen.getByRole('button', { name: 'Save' })
    expect(save).toBeDisabled()
    expect(screen.getByTestId('quote-unsaved')).toBeInTheDocument()

    fireEvent.change(screen.getByRole('textbox', { name: 'Quote name' }), { target: { value: 'Restaurant Sabor' } })
    fireEvent.click(save)

    await waitFor(() => expect(screen.getByTestId('quote-reference')).toHaveTextContent('Q-0001'))
    expect(quotesApi.createQuote).toHaveBeenCalledWith('tok', {
      name: 'Restaurant Sabor', discount_basis_points: 2000, currency_code: 'EUR',
      lines: [{ product_id: P1, sku_id: 's-fei', quantity: 10, unit_price_minor_units: 800 }],
    })
    expect(search).toContain('quote=q-1')
    expect(search).toContain(`lines=${P1}.s-fei.10.800`)
    await waitFor(() => expect(screen.queryByTestId('quote-unsaved')).not.toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('saves changes over the open quote, or as a new one', async () => {
    store['q-1'] = toSaved('q-1', 1, { name: 'Sabor', discount_basis_points: 0, currency_code: 'EUR',
      lines: [{ product_id: P1, sku_id: 's-fei', quantity: 10, unit_price_minor_units: 950 }] })
    nextNumber = 2
    renderPage(`/price-list?tab=quote&quote=q-1&lines=${P1}.s-fei.10.950`)
    await waitFor(() => expect(screen.getByRole('textbox', { name: 'Quote name' })).toHaveValue('Sabor'))
    const line = await screen.findByTestId('quote-line-s-fei')
    expect(screen.queryByTestId('quote-unsaved')).not.toBeInTheDocument()
    // The saved price differs from today's price, so today's is shown next to it.
    expect(within(line).getByText('Now')).toBeInTheDocument()

    fireEvent.change(within(line).getByRole('spinbutton', { name: 'Qty FEI-1' }), { target: { value: '12' } })
    await screen.findByTestId('quote-unsaved')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(quotesApi.updateQuote).toHaveBeenCalledWith('tok', 'q-1', expect.objectContaining({
      lines: [{ product_id: P1, sku_id: 's-fei', quantity: 12, unit_price_minor_units: 950 }],
    })))
    await waitFor(() => expect(screen.queryByTestId('quote-unsaved')).not.toBeInTheDocument())

    fireEvent.click(screen.getByRole('button', { name: 'Save as new' }))
    await waitFor(() => expect(screen.getByTestId('quote-reference')).toHaveTextContent('Q-0002'))
    expect(search).toContain('quote=q-2')
  })

  it('New quote empties the editor', async () => {
    store['q-1'] = toSaved('q-1', 1, { name: 'Sabor', discount_basis_points: 0, currency_code: 'EUR', lines: [] })
    renderPage(`/price-list?tab=quote&quote=q-1&lines=${P1}.s-fei.10.950`)
    await screen.findByTestId('quote-reference')

    fireEvent.click(screen.getByRole('button', { name: 'New quote' }))

    await waitFor(() => expect(screen.queryByTestId('quote-reference')).not.toBeInTheDocument())
    expect(screen.getByRole('textbox', { name: 'Quote name' })).toHaveValue('')
    expect(search).not.toContain('lines=')
    expect(search).not.toContain('quote=')
  })

  it('shows a failed save', async () => {
    vi.mocked(quotesApi.createQuote).mockRejectedValue(new Error('boom'))
    renderPage(`/price-list?tab=quote&lines=${P1}.s-fei.1`)
    await screen.findByTestId('quote-line-s-fei')
    fireEvent.change(screen.getByRole('textbox', { name: 'Quote name' }), { target: { value: 'x' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})

describe('saved quotes tab', () => {
  beforeEach(() => {
    store['q-1'] = toSaved('q-1', 1, { name: 'Restaurant Sabor', discount_basis_points: 1500, currency_code: 'EUR',
      lines: [{ product_id: P1, sku_id: 's-fei', quantity: 10, unit_price_minor_units: 850 }] })
  })

  it('lists quotes and opens one in the editor at its saved prices', async () => {
    renderPage('/price-list?tab=saved')
    const row = await screen.findByTestId('saved-quote-Q-0001')
    expect(row).toHaveTextContent('Restaurant Sabor')
    expect(row).toHaveTextContent('€85.00')
    expect(row).toHaveTextContent('15%')

    fireEvent.click(within(row).getByRole('button', { name: 'Open Q-0001' }))

    const line = await screen.findByTestId('quote-line-s-fei')
    expect(within(line).getByRole('textbox', { name: 'Unit price FEI-1' })).toHaveValue('8.50')
    expect(screen.getByRole('spinbutton', { name: 'Discount' })).toHaveValue(15)
    expect(search).toContain('tab=quote')
  })

  it('deletes after confirming, and an open copy of it becomes a new quote', async () => {
    renderPage(`/price-list?tab=saved&quote=q-1&lines=${P1}.s-fei.10.850`)
    const row = await screen.findByTestId('saved-quote-Q-0001')

    fireEvent.click(within(row).getByRole('button', { name: 'Delete Q-0001' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Delete quote' }))

    // URL updates commit in a router transition -- slow under a loaded run.
    await waitFor(() => expect(search).not.toContain('quote=q-1'))
    expect(await screen.findByText('No saved quotes yet.')).toBeInTheDocument()
    expect(quotesApi.deleteQuote).toHaveBeenCalledWith('tok', 'q-1')
  })

  it('cancelling the confirm keeps the quote', async () => {
    renderPage('/price-list?tab=saved')
    fireEvent.click(within(await screen.findByTestId('saved-quote-Q-0001')).getByRole('button', { name: 'Delete Q-0001' }))
    fireEvent.click(await screen.findByRole('button', { name: /cancel/i }))

    await waitFor(() => expect(screen.queryByRole('button', { name: 'Delete quote' })).not.toBeInTheDocument())
    expect(quotesApi.deleteQuote).not.toHaveBeenCalled()
  })
})
