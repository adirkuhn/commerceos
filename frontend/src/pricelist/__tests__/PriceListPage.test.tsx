// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import type { PriceListQuery, PriceListRow } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ listPriceList: vi.fn() }))
import * as api from '../api'
import PriceList from '../../pages/PriceList'

const P1 = '0b0e7e8e-0000-4000-8000-000000000001'
const P2 = '0b0e7e8e-0000-4000-8000-000000000002'

// Prices include 21% VAT: 1210 incl. = 1000 excl.; cost 500, target 50%.
const feijao: PriceListRow = {
  product_id: P1, product_name: 'Feijão', variant_name: '1kg', sku_id: 's-fei', sku_code: 'FEI-1', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 1210, sale_price_minor_units: 968, cost_price_minor_units: 500,
  available_quantity: 40, earliest_expiry: '2026-12-01', tax_rate_basis_points: 2100, target_markup_basis_points: 5000,
}
const farofa: PriceListRow = {
  product_id: P2, product_name: 'Farofa', variant_name: '', sku_id: 's-far', sku_code: 'FAR-1', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 605, sale_price_minor_units: null, cost_price_minor_units: null,
  available_quantity: 5, earliest_expiry: null, tax_rate_basis_points: 2100, target_markup_basis_points: 5000,
}
const ALL = [feijao, farofa]

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listPriceList).mockImplementation(async (_t: string, q: PriceListQuery) => {
    const rows = q.productIds ? ALL.filter(r => q.productIds!.includes(r.product_id))
      : ALL.filter(r => !q.search || r.product_name.toLowerCase().includes(q.search.toLowerCase()))
    return { rows, total: rows.length, pricesIncludeTax: true }
  })
})

let location = ''
function LocationProbe() {
  location = useLocation().search
  return null
}

function renderPage(url: string) {
  return render(<MemoryRouter initialEntries={[url]}><PriceList /><LocationProbe /></MemoryRouter>)
}

describe('price list tab', () => {
  it('shows price and sale price before and after the discount, with profit at the discounted price', async () => {
    renderPage('/price-list?discount=30')

    const row = await screen.findByTestId('price-row-s-fei')
    const cells = within(row).getAllByRole('cell').map(c => c.textContent)
    // 30% off: price 12.10 -> 8.47, sale 9.68 -> 6.78; profit on 6.78 incl. = 5.60 excl. - 5.00.
    expect(cells).toEqual(['Feijão — 1kgFEI-1', '40', '€5.00', '€12.10', '€8.47', '€9.68', '€6.78', '€0.60', '12.0%'])
    expect(row).toHaveClass('bg-danger/10')
  })

  it('a row with a healthy markup is not highlighted, and the discount comes from the field', async () => {
    renderPage('/price-list')
    const row = await screen.findByTestId('price-row-s-fei')
    expect(row).not.toHaveClass('bg-danger/10')

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Discount' }), { target: { value: '50' } })

    await waitFor(() => expect(screen.getByTestId('price-row-s-fei')).toHaveClass('bg-danger/10'))
    expect(location).toContain('discount=50')
  })

  it('shows a dash where cost is unknown', async () => {
    renderPage('/price-list')
    const row = await screen.findByTestId('price-row-s-far')
    expect(within(row).getAllByRole('cell')[2]).toHaveTextContent('—')
    expect(row).not.toHaveClass('bg-danger/10')
  })
})
