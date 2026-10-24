// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import type { PriceListQuery, PriceListRow } from '../../pricelist/types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../settings/company/api', () => ({ getCompanyProfile: vi.fn(async () => ({ legal_name: 'Cantinho B.V.', address: {} })) }))
vi.mock('../../inventory/api', () => ({ listWarehouses: vi.fn(async () => [{ id: 'wh-1', name: 'Amsterdam' }]) }))
vi.mock('../../oms/api', () => ({ listSalesOrders: vi.fn(async () => ({ salesOrders: [], total: 0 })) }))
vi.mock('../../pricelist/api', () => ({ listPriceList: vi.fn(), setAutomaticCostPrice: vi.fn() }))
vi.mock('../../newsletter/download', () => ({ downloadBlob: vi.fn() }))
import * as api from '../../pricelist/api'
import { downloadBlob } from '../../newsletter/download'
import Quotes from '../../pages/Quotes'

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
  return render(<MemoryRouter initialEntries={[url]}><Quotes /><LocationProbe /></MemoryRouter>)
}

describe('quote editor', () => {
  it('adds a searched product at the discounted price and totals the quote', async () => {
    renderPage('/quotes/new?discount=10')

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
    renderPage(`/quotes/new?lines=${P1}.s-fei.10.700,${P2}.s-far.8`)

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
    renderPage(`/quotes/new?lines=${P1}.s-fei.1`)
    const input = await screen.findByRole('textbox', { name: 'Unit price FEI-1' })

    fireEvent.change(input, { target: { value: '11,50' } })
    fireEvent.blur(input)
    await waitFor(() => expect(location).toContain(`${P1}.s-fei.1.1150`))

    fireEvent.click(screen.getByRole('button', { name: 'Back to the discounted price' }))
    await waitFor(() => expect(location).not.toContain('.1150'))
  })

  it('opens the quote document in a new tab, keeping the editor', async () => {
    const written: string[] = []
    const tab = { document: { open: vi.fn(), write: (h: string) => written.push(h), close: vi.fn() }, close: vi.fn() }
    const open = vi.spyOn(window, 'open').mockReturnValue(tab as never)
    renderPage(`/quotes/new?lines=${P1}.s-fei.3.999`)
    await screen.findByTestId('quote-line-s-fei')

    fireEvent.click(screen.getByRole('button', { name: 'Download PDF' }))

    expect(open).toHaveBeenCalledWith('', '_blank')
    await waitFor(() => expect(written[written.length - 1]).toContain('€29.97'))
    expect(screen.getByTestId('quote-line-s-fei')).toBeInTheDocument()
    open.mockRestore()
  })

  it('exports sku, product, quantity, unit price, line total and the order total', async () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true })
    renderPage(`/quotes/new?lines=${P1}.s-fei.3.999`)
    await screen.findByTestId('quote-line-s-fei')

    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }))

    expect(downloadBlob).toHaveBeenCalledTimes(1)
    const [blob, fileName] = vi.mocked(downloadBlob).mock.calls[0]
    expect(fileName).toMatch(/^quote-\d{4}-\d{2}-\d{2}\.csv$/)
    const bytes = await readBytes(blob)
    expect([...bytes.slice(0, 3)]).toEqual([0xef, 0xbb, 0xbf])
    expect(new TextDecoder().decode(bytes.slice(3))).toBe('SKU,Product,Quantity,Unit price,Line total\r\nFEI-1,Feijão — 1kg,3,9.99,29.97\r\n,,,Order total,29.97\r\n')
  })
})
