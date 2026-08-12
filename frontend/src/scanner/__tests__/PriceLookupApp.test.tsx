// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PriceLookupApp } from '../PriceLookupApp'
import type { SKU } from '../../pms/types'
import type { StockLevel } from '../../inventory/types'

vi.mock('../../pms/api', () => ({
  resolveBarcode: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  getSKUStockLevel: vi.fn(),
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

function stockLevel(overrides: Partial<StockLevel> = {}): StockLevel {
  return { sku_id: 'sku-1', total: 42, syncable: 42, ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
})

describe('PriceLookupApp', () => {
  it('resolves a scanned code and shows SKU code, price, and stock', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1 })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')

    expect(await screen.findByText('WID-L')).toBeInTheDocument()
    expect(screen.getByText('USD 19.99')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('shows the sale price alongside the regular price when one is active', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: sku({ sale_price_minor_units: 1499 }), units_per_scan: 1,
    })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')

    expect(await screen.findByText('USD 14.99')).toBeInTheDocument()
    expect(screen.getByText('USD 19.99')).toHaveClass('line-through')
  })

  it('shows an error when nothing matches', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'UNKNOWN{Enter}')

    expect(await screen.findByText('sku not found')).toBeInTheDocument()
  })

  it('Look up another returns to the scan screen', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1 })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')
    await screen.findByText('WID-L')

    await userEvent.click(screen.getByRole('button', { name: 'Look up another' }))

    expect(screen.getByPlaceholderText('Scan or type SKU / barcode…')).toBeInTheDocument()
  })

  it('resolves a code decoded via the camera', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1 })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} />)
    await userEvent.click(screen.getByRole('button', { name: /Use camera/ }))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'WID-L' }, undefined, { stop: vi.fn() }) })

    expect(await screen.findByText('WID-L')).toBeInTheDocument()
    expect(pmsApi.resolveBarcode).toHaveBeenCalledWith(TOKEN, 'WID-L')
  })
})
