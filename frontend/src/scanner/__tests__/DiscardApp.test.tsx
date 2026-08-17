// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DiscardApp } from '../DiscardApp'
import type { SKU, Product, ProductDetail } from '../../pms/types'
import type { Batch, StockLevel } from '../../inventory/types'

vi.mock('../../pms/api', () => ({
  resolveBarcode: vi.fn(),
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  listProductMedia: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  getSKUStockLevel: vi.fn(),
  listBatches: vi.fn(),
  discardBatch: vi.fn(),
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

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'product-1', tenant_id: 't1', name: 'Widget', slug: 'widget', description: '',
    status: 'active', has_variants: false, created_at: '', updated_at: '',
    ...overrides,
  } as Product
}

function productDetail(p: Product, skus: SKU[]): ProductDetail {
  return {
    product: p,
    variants: [{ id: `v-${p.id}`, tenant_id: 't1', product_id: p.id, name: 'Default', position: 0, created_at: '', updated_at: '', skus }],
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
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
})

async function resolveAndEnterSKU(matchedSKU = sku(), batches: Batch[] = [batch()], productName = 'Widget') {
  vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: matchedSKU, units_per_scan: 1, product_name: productName })
  vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
  vi.mocked(inventoryApi.listBatches).mockResolvedValue(batches)

  render(<DiscardApp token={TOKEN} onExit={() => {}} />)
  await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')
  await screen.findByText('WID-L')
}

describe('DiscardApp scan screen', () => {
  it('resolves a scanned code and shows the SKU screen with its batches', async () => {
    await resolveAndEnterSKU(sku(), [batch(), generalLedgerRow()])

    expect(screen.getByText('WID-L')).toBeInTheDocument()
    expect(screen.getByText('LOT-1')).toBeInTheDocument()
    expect(screen.getByText('General ledger')).toBeInTheDocument()
  })

  it('shows an error when nothing matches, even after the partial-match fallback', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    render(<DiscardApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'UNKNOWN{Enter}')

    expect(await screen.findByText('sku not found')).toBeInTheDocument()
  })

  it('falls back to a partial match and auto-resolves a single match', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))
    const p = product({ id: 'p1', name: 'Widget' })
    const s = sku({ id: 'sku-1', sku_code: 'WID-L' })
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [p], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue(productDetail(p, [s]))
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch()])

    render(<DiscardApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-{Enter}')

    expect(await screen.findByText('WID-L')).toBeInTheDocument()
  })

  it('resolves a code decoded via the camera', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1, product_name: 'Widget' })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch()])

    render(<DiscardApp token={TOKEN} onExit={() => {}} />)
    await userEvent.click(screen.getByRole('button', { name: /Use camera/ }))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'WID-L' }, undefined, { stop: vi.fn() }) })

    expect(await screen.findByText('WID-L')).toBeInTheDocument()
  })
})

describe('DiscardApp SKU screen', () => {
  it('Back returns to the scan screen', async () => {
    await resolveAndEnterSKU()

    await userEvent.click(screen.getByTitle('Back to scan'))

    expect(screen.getByPlaceholderText('Scan or type SKU / barcode…')).toBeInTheDocument()
  })

  it('tapping a row opens the discard sheet pre-filled with its full quantity, capped Max', async () => {
    await resolveAndEnterSKU(sku(), [batch({ quantity: 6 })])

    await userEvent.click(screen.getByText('LOT-1'))

    expect(screen.getByText('Discard LOT-1')).toBeInTheDocument()
    expect(screen.getByLabelText('Quantity to discard')).toHaveValue('6')
    await userEvent.type(screen.getByLabelText('Quantity to discard'), '99')
    expect(screen.getByLabelText('Quantity to discard')).toHaveValue('6')
  })

  it('Discard sends warehouse/batch/quantity/reason to the API and refreshes the list', async () => {
    vi.mocked(inventoryApi.discardBatch).mockResolvedValue({
      id: 'discard-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-1',
      movement_id: 'move-1', quantity: 6, reason: 'expired', discarded_by: '',
      discarded_at: '', created_at: '',
    })
    await resolveAndEnterSKU(sku(), [batch({ quantity: 6 })])

    await userEvent.click(screen.getByText('LOT-1'))
    await userEvent.clear(screen.getByLabelText('Quantity to discard'))
    await userEvent.type(screen.getByLabelText('Quantity to discard'), '4')
    await userEvent.type(screen.getByLabelText('Reason'), 'expired')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    await waitFor(() => expect(inventoryApi.discardBatch).toHaveBeenCalledWith(TOKEN, 'sku-1', {
      warehouse_id: 'wh-1', batch_id: 'batch-1', quantity: 4, reason: 'expired', photo: undefined,
    }))
    await waitFor(() => expect(inventoryApi.listBatches).toHaveBeenCalledTimes(2))
  })

  it('Discard on the general ledger row omits batch_id', async () => {
    vi.mocked(inventoryApi.discardBatch).mockResolvedValue({
      id: 'discard-2', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1',
      movement_id: 'move-2', quantity: 3, reason: '', discarded_by: '',
      discarded_at: '', created_at: '',
    })
    await resolveAndEnterSKU(sku(), [generalLedgerRow({ quantity: 20 })])

    await userEvent.click(screen.getByText('General ledger'))
    await userEvent.clear(screen.getByLabelText('Quantity to discard'))
    await userEvent.type(screen.getByLabelText('Quantity to discard'), '3')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    await waitFor(() => expect(inventoryApi.discardBatch).toHaveBeenCalledWith(TOKEN, 'sku-1', {
      warehouse_id: 'wh-1', batch_id: undefined, quantity: 3, reason: '', photo: undefined,
    }))
  })

  it('shows an error and keeps the sheet open on failure', async () => {
    vi.mocked(inventoryApi.discardBatch).mockRejectedValue(new Error('insufficient stock'))
    await resolveAndEnterSKU(sku(), [batch({ quantity: 6 })])

    await userEvent.click(screen.getByText('LOT-1'))
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(await screen.findByText('insufficient stock')).toBeInTheDocument()
    expect(screen.getByLabelText('Quantity to discard')).toBeInTheDocument()
  })
})
