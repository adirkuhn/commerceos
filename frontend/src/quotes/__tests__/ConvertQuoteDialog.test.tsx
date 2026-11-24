// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'

vi.mock('../quotesApi', () => ({ checkQuoteStock: vi.fn(), convertQuote: vi.fn() }))
vi.mock('../../customers/api', () => ({ listShippingAddresses: vi.fn() }))
import * as api from '../quotesApi'
import * as customersApi from '../../customers/api'
import type { ShippingAddress } from '../../customers/types'
import { ConvertQuoteDialog } from '../ConvertQuoteDialog'
import type { SalesOrder } from '../../oms/types'

const warehouses = [{ id: 'wh-1', name: 'Amsterdam' }, { id: 'wh-2', name: 'Utrecht' }]
const names = new Map([['s-1', 'Feijão'], ['s-2', 'Farofa']])
const created = { sales_order_id: 'so-1', number: 'B2B-1', needs_stock_review: false }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.convertQuote).mockResolvedValue(created)
})

function renderDialog(existing: Pick<SalesOrder, 'id' | 'number' | 'external_order_id'>[] = [], customerId: string | null = null) {
  const onCreated = vi.fn()
  const onClose = vi.fn()
  render(<ConvertQuoteDialog token="tok" quoteId="q-1" customerId={customerId} warehouses={warehouses} existingOrders={existing}
    productName={sku => names.get(sku) ?? sku} onClose={onClose} onCreated={onCreated} />)
  return { onCreated, onClose }
}

const book = (id: string, label: string, isDefault: boolean): ShippingAddress => ({
  id, customer_id: 'c-1', label, first_name: '', last_name: '', company: '', address_1: `${label} 1`, address_2: '',
  city: 'Utrecht', state: '', postcode: '', country: 'NL', phone: '', is_default: isDefault,
})

describe('ConvertQuoteDialog', () => {
  it("ships to the chosen one of the customer's addresses, the default preselected", async () => {
    vi.mocked(customersApi.listShippingAddresses).mockResolvedValue([book('sa-1', 'Home', true), book('sa-2', 'Warehouse', false)])
    vi.mocked(api.checkQuoteStock).mockResolvedValue({ has_shortfall: false, lines: [] })
    const { onCreated } = renderDialog([], 'c-1')

    const shipTo = await screen.findByRole('combobox', { name: 'Ship to' })
    expect(shipTo).toHaveValue('sa-1')
    fireEvent.change(shipTo, { target: { value: 'sa-2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create sales order' }))

    await waitFor(() => expect(onCreated).toHaveBeenCalled())
    expect(customersApi.listShippingAddresses).toHaveBeenCalledWith('tok', 'c-1')
    expect(api.convertQuote).toHaveBeenCalledWith('tok', 'q-1', { warehouse_id: 'wh-1', shipping_address_id: 'sa-2' })
  })

  it('has no address choice for a customer with one address', async () => {
    vi.mocked(customersApi.listShippingAddresses).mockResolvedValue([book('sa-1', 'Home', true)])
    renderDialog([], 'c-1')

    await waitFor(() => expect(customersApi.listShippingAddresses).toHaveBeenCalled())
    expect(screen.queryByRole('combobox', { name: 'Ship to' })).not.toBeInTheDocument()
  })

  it('creates the order straight away when the warehouse has everything', async () => {
    vi.mocked(api.checkQuoteStock).mockResolvedValue({ has_shortfall: false, lines: [{ product_id: 'p', sku_id: 's-1', quantity: 2, available: 2, short: 0 }] })
    const { onCreated } = renderDialog()

    fireEvent.change(screen.getByRole('combobox', { name: 'Warehouse' }), { target: { value: 'wh-2' } })
    fireEvent.change(screen.getByRole('textbox', { name: /order number/i }), { target: { value: 'B2B-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create sales order' }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith(created))
    expect(api.checkQuoteStock).toHaveBeenCalledWith('tok', 'q-1', 'wh-2')
    expect(api.convertQuote).toHaveBeenCalledWith('tok', 'q-1', { warehouse_id: 'wh-2', number: 'B2B-1' })
  })

  it('lists what is short and lets the user go back to fix the quote', async () => {
    vi.mocked(api.checkQuoteStock).mockResolvedValue({ has_shortfall: true, lines: [
      { product_id: 'p', sku_id: 's-1', quantity: 2, available: 2, short: 0 },
      { product_id: 'p', sku_id: 's-2', quantity: 10, available: 4, short: 6 },
    ] })
    const { onClose } = renderDialog()
    fireEvent.click(screen.getByRole('button', { name: 'Create sales order' }))

    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText('Farofa')).toBeInTheDocument()
    expect(within(alert).queryByText('Feijão')).not.toBeInTheDocument()
    expect(within(alert).getAllByRole('cell').map(c => c.textContent)).toEqual(['Farofa', '10', '4', '6'])
    expect(api.convertQuote).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Back to fix the quote' }))
    expect(onClose).toHaveBeenCalled()
  })

  it('creates anyway, the order going to Stock issues', async () => {
    vi.mocked(api.checkQuoteStock).mockResolvedValue({ has_shortfall: true, lines: [{ product_id: 'p', sku_id: 's-2', quantity: 10, available: 4, short: 6 }] })
    vi.mocked(api.convertQuote).mockResolvedValue({ ...created, needs_stock_review: true })
    const { onCreated } = renderDialog()
    fireEvent.click(screen.getByRole('button', { name: 'Create sales order' }))

    fireEvent.click(await screen.findByRole('button', { name: 'Create anyway' }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith({ ...created, needs_stock_review: true }))
  })

  it('a quote that already became an order needs two confirmations', async () => {
    vi.mocked(api.checkQuoteStock).mockResolvedValue({ has_shortfall: false, lines: [] })
    const { onCreated } = renderDialog([{ id: 'so-0', number: 'B2B-0', external_order_id: 'x' }])
    expect(screen.getByText(/already became/i)).toHaveTextContent('B2B-0')

    fireEvent.click(screen.getByRole('button', { name: 'Create sales order' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Create another order' }))
    expect(api.checkQuoteStock).not.toHaveBeenCalled()
    fireEvent.click(await screen.findByRole('button', { name: 'Yes, create a new order' }))

    await waitFor(() => expect(onCreated).toHaveBeenCalled())
  })

  it('shows a failed conversion', async () => {
    vi.mocked(api.checkQuoteStock).mockResolvedValue({ has_shortfall: false, lines: [] })
    vi.mocked(api.convertQuote).mockRejectedValue(new Error('boom'))
    renderDialog()
    fireEvent.click(screen.getByRole('button', { name: 'Create sales order' }))
    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})
