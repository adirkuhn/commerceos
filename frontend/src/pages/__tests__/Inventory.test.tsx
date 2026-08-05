// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Inventory from '../Inventory'
import type { Product, ProductDetail } from '../../pms/types'
import type { Warehouse, Batch, StockLevel, StockMovement } from '../../inventory/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  updateSKU: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  listBatches: vi.fn(),
  createBatch: vi.fn(),
  updateBatch: vi.fn(),
  deleteBatch: vi.fn(),
  getSKUStockLevel: vi.fn(),
  recordStockMovement: vi.fn(),
  recordRecount: vi.fn(),
  transferStock: vi.fn(),
  markDamagedOrLost: vi.fn(),
}))

import * as pmsApi from '../../pms/api'
import * as inventoryApi from '../../inventory/api'

const product: Product = {
  id: 'prod-1', tenant_id: 't1', name: 'Widget', slug: 'widget',
  description: '', status: 'active', has_variants: false, created_at: '', updated_at: '',
}

const sku = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'WIDGET-1',
  barcode: '', price_minor_units: 1000, currency_code: 'USD', is_virtual: false,
  status: 'active' as const, created_at: '', updated_at: '',
}

const detail: ProductDetail = {
  product,
  variants: [{ id: 'var-1', tenant_id: 't1', product_id: 'prod-1', name: 'Default', position: 0, created_at: '', updated_at: '', skus: [sku] }],
}

const warehouse: Warehouse = {
  id: 'wh-1', tenant_id: 't1', name: 'Main DC',
  address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
  created_at: '', updated_at: '',
}

const batch: Batch = {
  id: 'batch-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1',
  lot_number: 'LOT-1', expiry_date: null, received_at: '2026-01-01T00:00:00Z',
  created_at: '', updated_at: '',
}

const level: StockLevel = { sku_id: 'sku-1', total: 10, syncable: 10 }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product], total: 1 })
  vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse])
  vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch])
  vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(level)
})

describe('Inventory page', () => {
  it('lists products from the real API', async () => {
    render(<Inventory />)
    expect(await screen.findByText('Widget')).toBeInTheDocument()
  })

  it('shows price, sale price, and total stock in the collapsed product row', async () => {
    render(<Inventory />)

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(await screen.findByText('USD 10.00')).toBeInTheDocument()
    expect(await screen.findByText('10')).toBeInTheDocument()
    // No sale price on this fixture -- shown as a dash, not blank.
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
  })

  it('filters to only low-stock or out-of-stock products', async () => {
    const lowStockProduct: Product = { ...product, id: 'prod-2', name: 'Low Stock Widget', slug: 'low-stock-widget' }
    const outOfStockProduct: Product = { ...product, id: 'prod-3', name: 'Empty Widget', slug: 'empty-widget' }
    const lowSku = { ...sku, id: 'sku-2', sku_code: 'LOW-1' }
    const outSku = { ...sku, id: 'sku-3', sku_code: 'OUT-1' }

    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product, lowStockProduct, outOfStockProduct], total: 3 })
    vi.mocked(pmsApi.getProduct).mockImplementation(async (_token, id) => {
      if (id === 'prod-2') return { product: lowStockProduct, variants: [{ ...detail.variants[0], skus: [lowSku] }] }
      if (id === 'prod-3') return { product: outOfStockProduct, variants: [{ ...detail.variants[0], skus: [outSku] }] }
      return detail
    })
    vi.mocked(inventoryApi.getSKUStockLevel).mockImplementation(async (_token, skuId) => {
      if (skuId === 'sku-2') return { sku_id: skuId, total: 3, syncable: 3 } // <= threshold -- low
      if (skuId === 'sku-3') return { sku_id: skuId, total: 0, syncable: 0 } // zero -- out of stock
      return { sku_id: skuId, total: 50, syncable: 50 } // sku-1: well above the low-stock threshold
    })

    const user = userEvent.setup()
    render(<Inventory />)
    await screen.findByText('Widget')
    await screen.findByText('Low Stock Widget')
    await screen.findByText('Empty Widget')

    await user.click(screen.getByRole('button', { name: 'Low Stock' }))
    await waitFor(() => {
      expect(screen.getByText('Low Stock Widget')).toBeInTheDocument()
      expect(screen.queryByText('Widget')).not.toBeInTheDocument()
      expect(screen.queryByText('Empty Widget')).not.toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Out of Stock' }))
    await waitFor(() => {
      expect(screen.getByText('Empty Widget')).toBeInTheDocument()
      expect(screen.queryByText('Widget')).not.toBeInTheDocument()
      expect(screen.queryByText('Low Stock Widget')).not.toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'All' }))
    await waitFor(() => {
      expect(screen.getByText('Widget')).toBeInTheDocument()
      expect(screen.getByText('Low Stock Widget')).toBeInTheDocument()
      expect(screen.getByText('Empty Widget')).toBeInTheDocument()
    })
  })

  it('expanding a product loads its SKUs, stock level, and batches', async () => {
    const user = userEvent.setup()
    render(<Inventory />)

    await user.click(await screen.findByText('Widget'))

    expect(await screen.findByText('WIDGET-1')).toBeInTheDocument()
    expect(await screen.findByText('LOT-1')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByTestId('stock-level')).toHaveTextContent('Total: 10')
    })
  })

  it('recording a stock adjustment calls the API with the signed delta and refreshes the level', async () => {
    const movement: StockMovement = {
      id: 'mv-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: null,
      type: 'receipt', quantity_delta: 5, reason: '', occurred_at: '', created_at: '',
    }
    vi.mocked(inventoryApi.recordStockMovement).mockResolvedValue(movement)
    vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValueOnce(level).mockResolvedValue({ sku_id: 'sku-1', total: 15, syncable: 15 })

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /adjust/i }))
    const dialog = await screen.findByText('Adjust Stock')
    const modal = dialog.closest('div')!.parentElement!

    await user.type(within(modal).getByLabelText(/quantity/i), '5')
    await user.click(within(modal).getByRole('button', { name: /record/i }))

    await waitFor(() => {
      expect(inventoryApi.recordStockMovement).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
        warehouse_id: 'wh-1', type: 'receipt', quantity_delta: 5,
      }))
    })
  })

  it('surfaces an API error when creating a movement fails', async () => {
    vi.mocked(inventoryApi.recordStockMovement).mockRejectedValue(new Error('quantity_delta must be positive for receipt'))

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /adjust/i }))
    const dialog = await screen.findByText('Adjust Stock')
    const modal = dialog.closest('div')!.parentElement!
    await user.type(within(modal).getByLabelText(/quantity/i), '5')
    await user.click(within(modal).getByRole('button', { name: /record/i }))

    expect(await within(modal).findByText('quantity_delta must be positive for receipt')).toBeInTheDocument()
  })

  it('editing price sends the full existing SKU fields plus the new price/sale_price, and refreshes', async () => {
    vi.mocked(pmsApi.updateSKU).mockResolvedValue({ ...sku, price_minor_units: 1200, sale_price_minor_units: 900 })

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /^Price$/ }))
    const dialog = await screen.findByText('Edit Price')
    const modal = dialog.closest('div')!.parentElement!

    const priceInput = within(modal).getByLabelText(/^Regular price/)
    await user.clear(priceInput)
    await user.type(priceInput, '12')
    await user.type(within(modal).getByLabelText(/sale price/i), '9')
    await user.click(within(modal).getByRole('button', { name: /save/i }))

    await waitFor(() => {
      expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
        price_minor_units: 1200, sale_price_minor_units: 900, currency_code: 'USD', status: 'active',
      }))
    })
    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledTimes(2))
  })

  it('surfaces an API error when updating price fails', async () => {
    vi.mocked(pmsApi.updateSKU).mockRejectedValue(new Error('sale price cannot exceed regular price'))

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /^Price$/ }))
    const dialog = await screen.findByText('Edit Price')
    const modal = dialog.closest('div')!.parentElement!
    await user.click(within(modal).getByRole('button', { name: /save/i }))

    expect(await within(modal).findByText('sale price cannot exceed regular price')).toBeInTheDocument()
  })
})
