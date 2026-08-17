// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PriceLookupApp } from '../PriceLookupApp'
import type { SKU, Product, ProductDetail } from '../../pms/types'
import type { StockLevel } from '../../inventory/types'

vi.mock('../../pms/api', () => ({
  resolveBarcode: vi.fn(),
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  listProductMedia: vi.fn(),
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

beforeEach(() => {
  vi.clearAllMocks()
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
})

describe('PriceLookupApp', () => {
  it('resolves a scanned code and shows product name, SKU code, price, and stock', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1, product_name: 'Widget' })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('WID-L')).toBeInTheDocument()
    expect(screen.getByText('USD 19.99')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('View in Inventory hands off the resolved SKU and product name', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1, product_name: 'Widget' })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())
    const onViewInventory = vi.fn()

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={onViewInventory} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')
    await screen.findByText('Widget')
    await userEvent.click(screen.getByRole('button', { name: 'View in Inventory' }))

    expect(onViewInventory).toHaveBeenCalledWith({ sku: sku(), productName: 'Widget' })
  })

  it('shows the sale price alongside the regular price when one is active', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: sku({ sale_price_minor_units: 1499 }), units_per_scan: 1, product_name: 'Widget',
    })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')

    expect(await screen.findByText('USD 14.99')).toBeInTheDocument()
    expect(screen.getByText('USD 19.99')).toHaveClass('line-through')
  })

  it('shows an error when nothing matches, even after the partial-match fallback', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'UNKNOWN{Enter}')

    expect(await screen.findByText('sku not found')).toBeInTheDocument()
  })

  it('falls back to a partial match when the exact code lookup fails, auto-resolving a single match', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))
    const p = product({ id: 'p1', name: 'Widget' })
    const s = sku({ id: 'sku-1', sku_code: 'WID-L' })
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [p], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue(productDetail(p, [s]))
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-{Enter}')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('WID-L')).toBeInTheDocument()
  })

  it('shows a candidate list when the partial-match fallback finds more than one SKU', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockRejectedValue(new Error('sku not found'))
    const p1 = product({ id: 'p1', name: 'Widget' })
    const p2 = product({ id: 'p2', name: 'Widget Pro' })
    const s1 = sku({ id: 'sku-1', sku_code: 'WID-L' })
    const s2 = sku({ id: 'sku-2', sku_code: 'WID-PRO' })
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [p1, p2], total: 2 })
    vi.mocked(pmsApi.getProduct).mockImplementation(async (_token, id) =>
      id === 'p1' ? productDetail(p1, [s1]) : productDetail(p2, [s2]))
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID{Enter}')

    expect(await screen.findByText('Widget Pro')).toBeInTheDocument()
    await userEvent.click(screen.getByText('Widget Pro'))

    expect(await screen.findByText('WID-PRO')).toBeInTheDocument()
  })

  it('Look up another returns to the scan screen', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1, product_name: 'Widget' })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText('Scan or type SKU / barcode…'), 'WID-L{Enter}')
    await screen.findByText('WID-L')

    await userEvent.click(screen.getByRole('button', { name: 'Look up another' }))

    expect(screen.getByPlaceholderText('Scan or type SKU / barcode…')).toBeInTheDocument()
  })

  it('resolves a code decoded via the camera', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({ sku: sku(), units_per_scan: 1, product_name: 'Widget' })
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(stockLevel())

    render(<PriceLookupApp token={TOKEN} onExit={() => {}} onViewInventory={() => {}} />)
    await userEvent.click(screen.getByRole('button', { name: /Use camera/ }))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'WID-L' }, undefined, { stop: vi.fn() }) })

    expect(await screen.findByText('WID-L')).toBeInTheDocument()
    expect(pmsApi.resolveBarcode).toHaveBeenCalledWith(TOKEN, 'WID-L')
  })
})
