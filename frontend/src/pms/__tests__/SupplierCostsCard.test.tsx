// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SupplierCostsCard } from '../SupplierCostsCard'
import type { SKUSupplierCosts, SupplierCostEntry } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ listSKUSupplierCosts: vi.fn() }))
import * as api from '../api'

const row = (over: Partial<SupplierCostEntry>): SupplierCostEntry => ({
  kind: 'receipt', quantity: 10, unit_cost_minor_units: 650, landed_unit_cost_minor_units: 700, currency_code: 'EUR',
  received_at: '2026-10-20T09:00:00Z', purchase_order_id: 'po-2', purchase_order_line_item_id: 'l2',
  supplier_id: 'acme', supplier_name: 'Acme Foods', ...over,
})

const history = (over: Partial<SKUSupplierCosts> = {}): SKUSupplierCosts => ({
  currency_code: 'EUR',
  last: { landed_unit_cost_minor_units: 700, received_at: '2026-10-20T09:00:00Z', purchase_order_id: 'po-2', purchase_order_line_item_id: 'l2', supplier_id: 'acme', supplier_name: 'Acme Foods' },
  average_cost_minor_units: 600,
  by_supplier: [
    { supplier_id: 'acme', supplier_name: 'Acme Foods', last_landed_unit_cost_minor_units: 700, last_received_at: '2026-10-20T09:00:00Z', average_cost_minor_units: 600, quantity: 20, receipts: 2 },
    { supplier_id: 'bolt', supplier_name: 'Bolt Imports', last_landed_unit_cost_minor_units: 450, last_received_at: '2026-09-01T09:00:00Z', average_cost_minor_units: 450, quantity: 5, receipts: 1 },
  ],
  excluded_other_currency: 0,
  items: [row({}), row({ kind: 'reversal', quantity: -3, purchase_order_id: 'po-1', supplier_id: 'bolt', supplier_name: 'Bolt Imports' })],
  next_cursor: null,
  total: 2,
  ...over,
})

function renderCard(inventoryCost: number | null = 610) {
  return render(<MemoryRouter><SupplierCostsCard skuId="sku-1" inventoryCostMinorUnits={inventoryCost} /></MemoryRouter>)
}

beforeEach(() => vi.clearAllMocks())

describe('SupplierCostsCard', () => {
  it('shows the last receipt, the averages and each supplier', async () => {
    vi.mocked(api.listSKUSupplierCosts).mockResolvedValue(history())

    renderCard()

    const last = await screen.findByTestId('last-cost')
    expect(last).toHaveTextContent(/7[.,]00/)
    expect(last).toHaveTextContent('Acme Foods')
    expect(within(last).getByRole('link', { name: 'PO' })).toHaveAttribute('href', '/procurement/po-2')
    expect(screen.getByTestId('average-cost')).toHaveTextContent(/6[.,]00/)
    expect(screen.getByTestId('inventory-cost')).toHaveTextContent(/6[.,]10/)
    const suppliers = screen.getAllByTestId('supplier-row')
    expect(suppliers).toHaveLength(2)
    expect(suppliers[0]).toHaveTextContent('Acme Foods')
    expect(suppliers[1]).toHaveTextContent(/4[.,]50/)
    expect(api.listSKUSupplierCosts).toHaveBeenCalledWith('tok', 'sku-1', { supplierId: undefined, limit: 20 })
  })

  it('lists the history rows, a reversal marked as taken back', async () => {
    vi.mocked(api.listSKUSupplierCosts).mockResolvedValue(history())

    renderCard()

    const rows = await screen.findAllByTestId('history-row')
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveTextContent('Receipt')
    expect(rows[1]).toHaveTextContent('Taken back')
    expect(rows[1]).toHaveTextContent('-3')
  })

  it("filters the history to one supplier and back", async () => {
    vi.mocked(api.listSKUSupplierCosts).mockResolvedValue(history())
    const user = userEvent.setup()
    renderCard()

    await user.click(await screen.findByRole('button', { name: 'Bolt Imports' }))
    expect(api.listSKUSupplierCosts).toHaveBeenLastCalledWith('tok', 'sku-1', { supplierId: 'bolt', limit: 20 })

    await user.click(await screen.findByRole('button', { name: 'Only Bolt Imports · show all' }))
    expect(api.listSKUSupplierCosts).toHaveBeenLastCalledWith('tok', 'sku-1', { supplierId: undefined, limit: 20 })
  })

  it('loads the next page of history', async () => {
    vi.mocked(api.listSKUSupplierCosts)
      .mockResolvedValueOnce(history({ items: [row({})], next_cursor: 1, total: 2 }))
      .mockResolvedValueOnce(history({ items: [row({ purchase_order_id: 'po-1' })], next_cursor: null, total: 2 }))
    const user = userEvent.setup()
    renderCard()

    await user.click(await screen.findByRole('button', { name: 'Load more (1 of 2)' }))

    expect(api.listSKUSupplierCosts).toHaveBeenLastCalledWith('tok', 'sku-1', { supplierId: undefined, cursor: 1, limit: 20 })
    expect(await screen.findAllByTestId('history-row')).toHaveLength(2)
    expect(screen.queryByRole('button', { name: /Load more/ })).not.toBeInTheDocument()
  })

  it('says when receipts in another currency are left out', async () => {
    vi.mocked(api.listSKUSupplierCosts).mockResolvedValue(history({ excluded_other_currency: 2 }))

    renderCard()

    expect(await screen.findByText('2 receipts in another currency than EUR are left out of these figures.')).toBeInTheDocument()
  })

  it('names receipts without a supplier', async () => {
    vi.mocked(api.listSKUSupplierCosts).mockResolvedValue(history({
      by_supplier: [{ supplier_id: null, supplier_name: null, last_landed_unit_cost_minor_units: null, last_received_at: '2026-10-20T09:00:00Z', average_cost_minor_units: 0, quantity: 5, receipts: 1 }],
    }))

    renderCard(null)

    expect(within((await screen.findAllByTestId('supplier-row'))[0]).getByText('No supplier')).toBeInTheDocument()
    expect(screen.queryByTestId('inventory-cost')).not.toBeInTheDocument()
  })

  it('says when nothing was received yet', async () => {
    vi.mocked(api.listSKUSupplierCosts).mockResolvedValue(history({ last: null, average_cost_minor_units: null, by_supplier: [], items: [], total: 0 }))

    renderCard()

    expect(await screen.findByText('No goods received against a purchase order yet.')).toBeInTheDocument()
  })

  it('surfaces a load error', async () => {
    vi.mocked(api.listSKUSupplierCosts).mockRejectedValue(new Error('boom'))

    renderCard()

    expect(await screen.findByText(/boom|Failed to load the cost history/)).toBeInTheDocument()
  })
})
