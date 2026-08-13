// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { InventoryApp } from '../InventoryApp'
import type { SKU } from '../../pms/types'
import type { Batch, StockLevel, Warehouse } from '../../inventory/types'

vi.mock('../../pms/api', () => ({
  resolveBarcode: vi.fn(),
  updateSKU: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  getSKUStockLevel: vi.fn(),
  listBatches: vi.fn(),
  listWarehouses: vi.fn(),
  recordStockMovement: vi.fn(),
  transferStock: vi.fn(),
  markDamagedOrLost: vi.fn(),
  createBatch: vi.fn(),
}))

const mockDecodeFromVideoDevice = vi.fn()
vi.mock('@zxing/browser', () => ({
  BrowserMultiFormatReader: vi.fn().mockImplementation(() => ({
    decodeFromVideoDevice: mockDecodeFromVideoDevice,
  })),
}))

import * as pmsApi from '../../pms/api'
import * as inventoryApi from '../../inventory/api'

const TOKEN = 'test-token'

function sku(overrides: Partial<SKU> = {}): SKU {
  return {
    id: 'sku-1', tenant_id: 't1', variant_id: 'v1', sku_code: 'WID-L',
    price_minor_units: 1999, currency_code: 'USD', is_virtual: false, status: 'active',
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function warehouse(overrides: Partial<Warehouse> = {}): Warehouse {
  return {
    id: 'wh-1', tenant_id: 't1', name: 'Main Warehouse',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function batch(overrides: Partial<Batch> = {}): Batch {
  return {
    id: 'batch-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1',
    lot_number: 'LOT-1', expiry_date: null, received_at: '', created_at: '', updated_at: '',
    quantity: 10, is_general_ledger: false,
    ...overrides,
  }
}

function generalLedgerRow(overrides: Partial<Batch> = {}): Batch {
  return batch({ id: 'gl-wh-1', lot_number: '', is_general_ledger: true, quantity: 20, ...overrides })
}

function stockLevel(overrides: Partial<StockLevel> = {}): StockLevel {
  return { sku_id: 'sku-1', total: 10, syncable: 10, ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
})

async function resolveAndEnterSKU(matchedSKU = sku(), batches: Batch[] = [batch()], productName = 'Widget') {
  vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: matchedSKU, units_per_scan: 1, product_name: productName })
  vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
  vi.mocked(inventoryApi.listBatches).mockResolvedValue(batches)

  render(<InventoryApp token={TOKEN} onExit={() => {}} />)
  await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')
  await screen.findByText('WID-L')
}

describe('InventoryApp scan screen', () => {
  it('resolves a scanned code and shows the SKU screen', async () => {
    await resolveAndEnterSKU()

    expect(screen.getByText('WID-L')).toBeInTheDocument()
    expect(await screen.findByText('LOT-1')).toBeInTheDocument()
    expect(screen.getByText('Main Warehouse')).toBeInTheDocument()
  })

  it('shows the product name and SKU code (small) as the heading, and total stock below', async () => {
    await resolveAndEnterSKU(sku({ sku_code: 'WID-L' }), [batch()], 'Widget')

    const heading = screen.getByRole('heading')
    expect(heading).toHaveTextContent('Widget')
    expect(heading).toHaveTextContent('WID-L')
    expect(screen.getByText('Total stock: 10')).toBeInTheDocument()
  })

  it('shows the regular price on the Price button when there is no sale price', async () => {
    await resolveAndEnterSKU(sku({ price_minor_units: 1999, currency_code: 'USD' }))

    expect(screen.getByRole('button', { name: /Price/ })).toHaveTextContent('USD 19.99')
  })

  it('shows the sale price with the original struck through on the Price button when a sale is active', async () => {
    await resolveAndEnterSKU(sku({ price_minor_units: 1999, sale_price_minor_units: 1499, currency_code: 'USD' }))

    const priceButton = screen.getByRole('button', { name: /Price/ })
    expect(priceButton).toHaveTextContent('USD 14.99')
    expect(priceButton).toHaveTextContent('USD 19.99')
  })

  it('shows an error when nothing matches', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    render(<InventoryApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'UNKNOWN{Enter}')

    expect(await screen.findByText('sku not found')).toBeInTheDocument()
  })

  it('resolves a code decoded via the camera', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1, product_name: 'Widget' })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch()])

    render(<InventoryApp token={TOKEN} onExit={() => {}} />)
    await userEvent.click(screen.getByRole('button', { name: /Use camera/ }))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'WID-L' }, undefined, { stop: vi.fn() }) })

    expect(await screen.findByText('WID-L')).toBeInTheDocument()
    expect(pmsApi.resolveBarcode).toHaveBeenCalledWith(TOKEN, 'WID-L')
  })
})

describe('InventoryApp SKU screen', () => {
  it('Back returns to the scan screen', async () => {
    await resolveAndEnterSKU()

    await userEvent.click(screen.getByTitle('Back to scan'))

    expect(screen.getByPlaceholderText('Scan or type SKU / barcode…')).toBeInTheDocument()
  })

  it('tapping a batch row opens the action sheet scoped to that row', async () => {
    await resolveAndEnterSKU(sku(), [batch(), generalLedgerRow()])

    await userEvent.click(screen.getByText('LOT-1'))

    expect(screen.getAllByText('LOT-1').length).toBeGreaterThan(1) // row label + sheet header
    expect(screen.getByRole('button', { name: /Adjust/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Move/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Damaged.*Lost/ })).toBeInTheDocument()
  })

  it('Adjust records a stock movement scoped to the tapped batch, with the entered quantity and reason', async () => {
    vi.mocked(inventoryApi.recordStockMovement).mockResolvedValue({
      id: 'move-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-1', type: 'adjustment',
      quantity_delta: 5, reason: 'recount', occurred_at: '', created_at: '',
    })
    await resolveAndEnterSKU(sku(), [batch(), generalLedgerRow()])

    await userEvent.click(screen.getByText('LOT-1'))
    await userEvent.click(screen.getByRole('button', { name: /Adjust/ }))
    await userEvent.type(screen.getByLabelText('Quantity change'), '5')
    await userEvent.type(screen.getByLabelText('Reason'), 'recount')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.recordStockMovement).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      warehouse_id: 'wh-1', batch_id: 'batch-1', type: 'adjustment', quantity_delta: 5, reason: 'recount',
    })))
  })

  it('Adjust on the general ledger row is scoped with batch_id null, not the first real batch', async () => {
    vi.mocked(inventoryApi.recordStockMovement).mockResolvedValue({
      id: 'move-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'adjustment',
      quantity_delta: 3, reason: '', occurred_at: '', created_at: '',
    })
    await resolveAndEnterSKU(sku(), [batch(), generalLedgerRow()])

    await userEvent.click(screen.getByText('General ledger'))
    await userEvent.click(screen.getByRole('button', { name: /Adjust/ }))
    await userEvent.type(screen.getByLabelText('Quantity change'), '3')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.recordStockMovement).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      warehouse_id: 'wh-1', batch_id: null, quantity_delta: 3,
    })))
  })

  it('Move transfers stock from the tapped batch to a different warehouse', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse(), warehouse({ id: 'wh-2', name: 'Overflow' })])
    vi.mocked(inventoryApi.transferStock).mockResolvedValue({
      out: { id: 'm1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'transfer_out', quantity_delta: -3, reason: '', occurred_at: '', created_at: '' },
      in: { id: 'm2', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-2', type: 'transfer_in', quantity_delta: 3, reason: '', occurred_at: '', created_at: '' },
    })
    await resolveAndEnterSKU(sku(), [batch()])

    await userEvent.click(screen.getByText('LOT-1'))
    await userEvent.click(screen.getByRole('button', { name: /Move/ }))
    await userEvent.selectOptions(screen.getByLabelText('To warehouse'), 'wh-2')
    await userEvent.type(screen.getByLabelText('Quantity to move'), '3')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.transferStock).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      from_warehouse_id: 'wh-1', from_batch_id: 'batch-1', to_warehouse_id: 'wh-2', to_batch_id: null, quantity: 3,
    })))
  })

  it('Move can target an existing batch as the destination', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.transferStock).mockResolvedValue({
      out: { id: 'm1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'transfer_out', quantity_delta: -2, reason: '', occurred_at: '', created_at: '' },
      in: { id: 'm2', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-2', type: 'transfer_in', quantity_delta: 2, reason: '', occurred_at: '', created_at: '' },
    })
    await resolveAndEnterSKU(sku(), [batch(), batch({ id: 'batch-2', lot_number: 'LOT-2', warehouse_id: 'wh-1' })])

    await userEvent.click(screen.getByText('LOT-1'))
    await userEvent.click(screen.getByRole('button', { name: /Move/ }))
    // "LOT-2" appears both as the underlying row (still rendered behind the
    // sheet) and as the sheet's own destination-batch option -- the sheet's
    // is the one that renders last in DOM order.
    const lot2Options = screen.getAllByText('LOT-2')
    await userEvent.click(lot2Options[lot2Options.length - 1])
    await userEvent.type(screen.getByLabelText('Quantity to move'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.transferStock).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      from_batch_id: 'batch-1', to_warehouse_id: 'wh-1', to_batch_id: 'batch-2', quantity: 2,
    })))
  })

  it('Move to "+ New batch" creates the batch first, then transfers into it', async () => {
    vi.mocked(inventoryApi.createBatch).mockResolvedValue(batch({ id: 'batch-new', lot_number: 'LOT-NEW' }))
    vi.mocked(inventoryApi.transferStock).mockResolvedValue({
      out: { id: 'm1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'transfer_out', quantity_delta: -4, reason: '', occurred_at: '', created_at: '' },
      in: { id: 'm2', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-new', type: 'transfer_in', quantity_delta: 4, reason: '', occurred_at: '', created_at: '' },
    })
    await resolveAndEnterSKU(sku(), [generalLedgerRow()])

    await userEvent.click(screen.getByText('General ledger'))
    await userEvent.click(screen.getByRole('button', { name: /Move/ }))
    await userEvent.click(screen.getByText('+ New batch'))
    await userEvent.type(screen.getByLabelText('Lot number'), 'LOT-NEW')
    await userEvent.type(screen.getByLabelText('Quantity to move'), '4')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.createBatch).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      warehouse_id: 'wh-1', lot_number: 'LOT-NEW',
    })))
    await waitFor(() => expect(inventoryApi.transferStock).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      from_warehouse_id: 'wh-1', from_batch_id: null, to_warehouse_id: 'wh-1', to_batch_id: 'batch-new', quantity: 4,
    })))
  })

  it('Damaged/Lost records a loss movement scoped to the tapped batch', async () => {
    vi.mocked(inventoryApi.markDamagedOrLost).mockResolvedValue({
      id: 'move-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-1', type: 'damaged',
      quantity_delta: -2, reason: 'dropped', occurred_at: '', created_at: '',
    })
    await resolveAndEnterSKU(sku(), [batch()])

    await userEvent.click(screen.getByText('LOT-1'))
    await userEvent.click(screen.getByRole('button', { name: /Damaged.*Lost/ }))
    await userEvent.type(screen.getByLabelText('Quantity'), '2')
    await userEvent.type(screen.getByLabelText('Reason'), 'dropped')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.markDamagedOrLost).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      warehouse_id: 'wh-1', batch_id: 'batch-1', quantity: 2, type: 'damaged', reason: 'dropped',
    })))
  })

  it('Price edits the SKU price, round-tripping every other field so nothing gets silently wiped', async () => {
    const mySKU = sku({
      barcode: '012345', price_minor_units: 1999, currency_code: 'USD',
      weight_grams: 500, is_virtual: false, tracks_expiry: true, unit_type: 'unit', status: 'active',
    })
    vi.mocked(pmsApi.updateSKU).mockResolvedValue({ ...mySKU, price_minor_units: 2499 })
    await resolveAndEnterSKU(mySKU, [batch()])

    await userEvent.click(screen.getByRole('button', { name: /Price/ }))
    await userEvent.clear(screen.getByLabelText(/Regular price/))
    await userEvent.type(screen.getByLabelText(/Regular price/), '24.99')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      barcode: '012345', price_minor_units: 2499, currency_code: 'USD',
      // Fields the price sheet itself never edits -- must still round-trip
      // unchanged, not silently reset to their zero-value default.
      weight_grams: 500, is_virtual: false, tracks_expiry: true, unit_type: 'unit', status: 'active',
    })))
  })

  it('Price shows an error and keeps the sheet open on failure', async () => {
    vi.mocked(pmsApi.updateSKU).mockRejectedValue(new Error('price update failed'))
    await resolveAndEnterSKU(sku(), [batch()])

    await userEvent.click(screen.getByRole('button', { name: /Price/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('price update failed')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })
})
