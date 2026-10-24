// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import type { PriceListQuery, PriceListRow } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ listPriceList: vi.fn() }))
vi.mock('../../newsletter/download', () => ({ downloadBlob: vi.fn() }))
import * as api from '../api'
import { downloadBlob } from '../../newsletter/download'
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

// jsdom's Blob has no arrayBuffer().
function readBytes(blob: Blob): Promise<Uint8Array> {
  return new Promise(resolve => {
    const reader = new FileReader()
    reader.onload = () => resolve(new Uint8Array(reader.result as ArrayBuffer))
    reader.readAsArrayBuffer(blob)
  })
}

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

describe('B2B quote tab', () => {
  it('adds a searched product at the discounted price and totals the quote', async () => {
    renderPage('/price-list?tab=quote&discount=10')

    fireEvent.change(screen.getByRole('textbox', { name: /search products/i }), { target: { value: 'fei' } })
    fireEvent.click(await screen.findByRole('button', { name: /add feijão FEI-1/i }))

    const line = await screen.findByTestId('quote-line-s-fei')
    // 10% off the 9.68 sale price.
    expect(within(line).getByRole('textbox', { name: 'Unit price FEI-1' })).toHaveValue('8.71')
    expect(within(line).getAllByRole('cell')[1]).toHaveTextContent('2026')

    fireEvent.change(within(line).getByRole('spinbutton', { name: 'Qty FEI-1' }), { target: { value: '100' } })

    await waitFor(() => expect(screen.getByTestId('quote-total')).toHaveTextContent('€871.00'))
    // 8.71 incl. = 7.20 excl. -> 2.20 profit a unit.
    expect(screen.getByTestId('quote-profit')).toHaveTextContent('€220.00')
    expect(location).toContain(`lines=${P1}.s-fei.100`)
  })

  it('reloads the quote from the URL and flags a typed price under the target', async () => {
    renderPage(`/price-list?tab=quote&lines=${P1}.s-fei.10.700,${P2}.s-far.8`)

    const line = await screen.findByTestId('quote-line-s-fei')
    expect(api.listPriceList).toHaveBeenCalledWith('tok', expect.objectContaining({ productIds: [P1, P2] }))
    expect(line).toHaveClass('bg-danger/10')
    expect(within(line).getByRole('button', { name: 'Back to the discounted price' })).toBeInTheDocument()

    const farofaLine = screen.getByTestId('quote-line-s-far')
    expect(within(farofaLine).getAllByRole('cell')[2]).toHaveAttribute('title', 'Only 5 available')
    // 10 x 7.00 + 8 x 6.05; profit only from the line with a cost.
    expect(screen.getByTestId('quote-total')).toHaveTextContent('€118.40')
    expect(screen.getByTestId('quote-profit')).toHaveTextContent('*')
  })

  it('typing a price over the discounted one, then resetting it', async () => {
    renderPage(`/price-list?tab=quote&lines=${P1}.s-fei.1`)
    const input = await screen.findByRole('textbox', { name: 'Unit price FEI-1' })

    fireEvent.change(input, { target: { value: '11,50' } })
    fireEvent.blur(input)
    await waitFor(() => expect(location).toContain(`${P1}.s-fei.1.1150`))

    fireEvent.click(screen.getByRole('button', { name: 'Back to the discounted price' }))
    await waitFor(() => expect(location).not.toContain('.1150'))
  })

  it('opens the printable quote for Save as PDF', async () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    renderPage(`/price-list?tab=quote&lines=${P1}.s-fei.3.999`)
    await screen.findByTestId('quote-line-s-fei')

    fireEvent.click(screen.getByRole('button', { name: 'Download PDF' }))

    expect(print).toHaveBeenCalled()
    expect(screen.getByRole('heading', { name: 'Quote' })).toBeInTheDocument()
    expect(screen.getAllByText('€29.97', { selector: 'td' })).toHaveLength(2) // line total and order total
    print.mockRestore()
  })

  it('exports product, quantity and unit price only', async () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true })
    renderPage(`/price-list?tab=quote&lines=${P1}.s-fei.3.999`)
    await screen.findByTestId('quote-line-s-fei')

    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }))

    expect(downloadBlob).toHaveBeenCalledTimes(1)
    const [blob, fileName] = vi.mocked(downloadBlob).mock.calls[0]
    expect(fileName).toMatch(/^quote-\d{4}-\d{2}-\d{2}\.csv$/)
    const bytes = await readBytes(blob)
    expect([...bytes.slice(0, 3)]).toEqual([0xef, 0xbb, 0xbf])
    expect(new TextDecoder().decode(bytes.slice(3))).toBe('Product,Quantity,Unit price\r\nFeijão — 1kg (FEI-1),3,9.99\r\n')
  })
})
