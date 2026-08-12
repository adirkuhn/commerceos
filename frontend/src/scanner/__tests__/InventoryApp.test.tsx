// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { InventoryApp } from '../InventoryApp'
import type { SKU } from '../../pms/types'
import type { Batch, StockLevel, Warehouse } from '../../inventory/types'

vi.mock('../../pms/api', () => ({
  resolveBarcode: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  getSKUStockLevel: vi.fn(),
  listBatches: vi.fn(),
  listWarehouses: vi.fn(),
  recordStockMovement: vi.fn(),
  transferStock: vi.fn(),
  markDamagedOrLost: vi.fn(),
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

function stockLevel(overrides: Partial<StockLevel> = {}): StockLevel {
  return { sku_id: 'sku-1', total: 10, syncable: 10, ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
})

async function resolveAndEnterSKU(matchedSKU = sku()) {
  vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: matchedSKU, units_per_scan: 1 })
  vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
  vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch()])

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

  it('shows an error when nothing matches', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    render(<InventoryApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'UNKNOWN{Enter}')

    expect(await screen.findByText('sku not found')).toBeInTheDocument()
  })

  it('resolves a code decoded via the camera', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1 })
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

  it('Adjust records a stock movement with the entered quantity and reason', async () => {
    vi.mocked(inventoryApi.recordStockMovement).mockResolvedValue({
      id: 'move-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'adjustment',
      quantity_delta: 5, reason: 'recount', occurred_at: '', created_at: '',
    })
    await resolveAndEnterSKU()

    await userEvent.click(screen.getByRole('button', { name: /Adjust/ }))
    await userEvent.type(screen.getByLabelText('Quantity change'), '5')
    await userEvent.type(screen.getByLabelText('Reason'), 'recount')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm adjustment' }))

    await waitFor(() => expect(inventoryApi.recordStockMovement).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      warehouse_id: 'wh-1', type: 'adjustment', quantity_delta: 5, reason: 'recount',
    })))
  })

  it('Move transfers stock to a different warehouse', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse(), warehouse({ id: 'wh-2', name: 'Overflow' })])
    vi.mocked(inventoryApi.transferStock).mockResolvedValue({
      out: { id: 'm1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'transfer_out', quantity_delta: -3, reason: '', occurred_at: '', created_at: '' },
      in: { id: 'm2', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-2', type: 'transfer_in', quantity_delta: 3, reason: '', occurred_at: '', created_at: '' },
    })
    await resolveAndEnterSKU()

    await userEvent.click(screen.getByRole('button', { name: /Move/ }))
    await userEvent.selectOptions(screen.getByLabelText('To warehouse'), 'wh-2')
    await userEvent.type(screen.getByLabelText('Quantity to move'), '3')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm move' }))

    await waitFor(() => expect(inventoryApi.transferStock).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      from_warehouse_id: 'wh-1', to_warehouse_id: 'wh-2', quantity: 3,
    })))
  })

  it('Damaged/Lost records a loss movement', async () => {
    vi.mocked(inventoryApi.markDamagedOrLost).mockResolvedValue({
      id: 'move-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'damaged',
      quantity_delta: -2, reason: 'dropped', occurred_at: '', created_at: '',
    })
    await resolveAndEnterSKU()

    await userEvent.click(screen.getByRole('button', { name: /Damaged.*Lost/ }))
    await userEvent.type(screen.getByLabelText('Quantity'), '2')
    await userEvent.type(screen.getByLabelText('Reason'), 'dropped')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.markDamagedOrLost).toHaveBeenCalledWith(TOKEN, 'sku-1', expect.objectContaining({
      warehouse_id: 'wh-1', quantity: 2, type: 'damaged', reason: 'dropped',
    })))
  })
})
