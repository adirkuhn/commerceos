// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Inventory from '../Inventory'
import type { Product, ProductDetail } from '../../pms/types'
import type { Warehouse, Batch, StockLevel, StockMovement, ExpiredBatchReview } from '../../inventory/types'

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
  listStockMovements: vi.fn(),
  recordRecount: vi.fn(),
  transferStock: vi.fn(),
  splitCaseIntoUnits: vi.fn(),
  portionFromBulk: vi.fn(),
  markDamagedOrLost: vi.fn(),
  listPendingExpiredBatches: vi.fn(),
  writeOffExpiredBatch: vi.fn(),
  extendBatchExpiry: vi.fn(),
  dismissExpiredBatch: vi.fn(),
  listBatchMovements: vi.fn(),
  listExpiringBatches: vi.fn(),
  discardBatch: vi.fn(),
  listDiscards: vi.fn(),
}))

import * as pmsApi from '../../pms/api'
import * as inventoryApi from '../../inventory/api'

const product: Product = {
  id: 'prod-1', tenant_id: 't1', name: 'Widget', slug: 'widget',
  description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '',
}

const sku = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'WIDGET-1',
  barcode: '', price_minor_units: 1000, currency_code: 'USD', is_virtual: false,
  // Non-default sentinel -- the price-edit regression test proves this
  // round-trips through an update instead of getting silently reset, since
  // UpdateSKU is a full replace, not a patch. (Deliberately not unit_type:
  // 'kilo' here -- that would also change how other tests in this file
  // render quantity, since they share this fixture.)
  tracks_expiry: true,
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
  // Non-zero: a zero-stock batch is hidden by default in the table (see
  // the "Hide zero-stock batches" describe block), which most tests in
  // this file aren't testing for, so the shared fixture needs stock.
  quantity: 10, is_general_ledger: false,
}

const level: StockLevel = { sku_id: 'sku-1', total: 10, syncable: 10 }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product], total: 1 })
  vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse])
  vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch])
  vi.mocked(inventoryApi.getSKUStockLevel).mockResolvedValue(level)
  vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([])
  vi.mocked(inventoryApi.listExpiringBatches).mockResolvedValue([])
  vi.mocked(inventoryApi.listDiscards).mockResolvedValue({ discards: [], total: 0 })
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

  it('shows the item count per batch', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([{ ...batch, quantity: 82 }])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('LOT-1')

    const row = (await screen.findByText('LOT-1')).closest('tr')!
    expect(within(row).getByText('82')).toBeInTheDocument()
  })

  it('hides a zero-stock batch by default and shows it after clicking Show', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([
      { ...batch, id: 'batch-empty', lot_number: 'LOT-EMPTY', quantity: 0 },
      { ...batch, id: 'batch-2', lot_number: 'LOT-2', quantity: 5 },
    ])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('LOT-2')

    expect(screen.queryByText('LOT-EMPTY')).not.toBeInTheDocument()
    const toggle = screen.getByRole('button', { name: 'Show 1 zero-stock batch' })

    await user.click(toggle)

    expect(await screen.findByText('LOT-EMPTY')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Hide zero-stock batches' })).toBeInTheDocument()
  })

  it('the general ledger row is never hidden even at zero quantity', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([
      { ...batch, id: '', lot_number: '', is_general_ledger: true, quantity: 0 },
    ])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))

    expect(await screen.findByText('General ledger')).toBeInTheDocument()
    expect(screen.queryByText(/zero-stock/)).not.toBeInTheDocument()
  })

  it('shows a general ledger row first, ahead of real batches, with no edit/delete actions', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([
      { ...batch, id: '', lot_number: '', is_general_ledger: true, quantity: 0 },
      { ...batch, quantity: 82 },
    ])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('LOT-1')

    await screen.findByText('General ledger')
    const qtyHeader = screen.getByText('Qty')
    const batchTable = qtyHeader.closest('table')!
    const rows = within(batchTable).getAllByRole('row').slice(1) // drop header row
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('General ledger')
    expect(rows[1].textContent).toContain('LOT-1')
    expect(within(rows[0]).queryByLabelText(/Edit batch/)).not.toBeInTheDocument()
    expect(within(rows[0]).queryByLabelText(/Delete batch/)).not.toBeInTheDocument()
  })

  it('surfaces an insufficient-stock error instead of failing silently', async () => {
    vi.mocked(inventoryApi.recordStockMovement).mockRejectedValue(new Error('insufficient stock for this batch/ledger'))

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /adjust/i }))
    const dialog = await screen.findByText('Adjust Stock')
    const modal = dialog.closest('div')!.parentElement!
    await user.type(within(modal).getByLabelText(/quantity/i), '5')
    await user.click(within(modal).getByRole('button', { name: /record/i }))

    expect(await within(modal).findByText('insufficient stock for this batch/ledger')).toBeInTheDocument()
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
        // A field the price sheet itself never edits -- must still
        // round-trip unchanged, not silently reset to its zero-value default.
        tracks_expiry: true,
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

  it('does not show a Split case action for an ordinary SKU', async () => {
    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    expect(screen.queryByRole('button', { name: /Split case/i })).not.toBeInTheDocument()
  })

  it('splits case stock into units for a SKU configured as a case of another product', async () => {
    const caseSku = { ...sku, case_of_sku_id: 'sku-unit-1', units_per_case: 36 }
    const caseDetail: ProductDetail = {
      product,
      variants: [{ ...detail.variants[0], skus: [caseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(caseDetail)
    vi.mocked(inventoryApi.splitCaseIntoUnits).mockResolvedValue({
      case_out: { id: 'mv-out', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: null, type: 'split_out', quantity_delta: -2, reason: '', occurred_at: '', created_at: '' },
      unit_in: { id: 'mv-in', tenant_id: 't1', sku_id: 'sku-unit-1', warehouse_id: 'wh-1', batch_id: null, type: 'split_in', quantity_delta: 72, reason: '', occurred_at: '', created_at: '' },
    })

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /Split case/i }))
    const dialog = await screen.findByText('Split Case Into Units')
    const modal = dialog.closest('div')!.parentElement!
    await user.type(within(modal).getByLabelText(/cases to split/i), '2')
    await user.click(within(modal).getByRole('button', { name: /^Split$/ }))

    await waitFor(() => {
      expect(inventoryApi.splitCaseIntoUnits).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
        warehouse_id: 'wh-1', cases_to_split: 2,
      }))
    })
  })

  it('does not show an Add item from weight pool action for an ordinary SKU', async () => {
    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    expect(screen.queryByRole('button', { name: /Add item from weight pool/i })).not.toBeInTheDocument()
  })

  it('portions a sellable item off a bulk SKU\'s weight pool, defaulting the weight from the SKU', async () => {
    const portionedSku = { ...sku, portioned_from_sku_id: 'sku-bulk-1', weight_grams: 378 }
    const portionedDetail: ProductDetail = {
      product,
      variants: [{ ...detail.variants[0], skus: [portionedSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(portionedDetail)
    vi.mocked(inventoryApi.portionFromBulk).mockResolvedValue({
      bulk_out: { id: 'mv-out', tenant_id: 't1', sku_id: 'sku-bulk-1', warehouse_id: 'wh-1', batch_id: null, type: 'split_out', quantity_delta: -378, reason: '', occurred_at: '', created_at: '' },
      unit_in: { id: 'mv-in', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: null, type: 'split_in', quantity_delta: 1, reason: '', occurred_at: '', created_at: '' },
    })

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /Add item from weight pool/i }))
    const dialog = await screen.findByText('Add Item From Weight Pool')
    const modal = dialog.closest('div')!.parentElement!
    await user.click(within(modal).getByRole('button', { name: /^Add item$/ }))

    await waitFor(() => {
      expect(inventoryApi.portionFromBulk).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
        warehouse_id: 'wh-1', weight_grams: undefined,
      }))
    })
  })

  it('portions with an explicit scale-reading weight overriding the nominal weight', async () => {
    const portionedSku = { ...sku, portioned_from_sku_id: 'sku-bulk-1', weight_grams: 400 }
    const portionedDetail: ProductDetail = {
      product,
      variants: [{ ...detail.variants[0], skus: [portionedSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(portionedDetail)
    vi.mocked(inventoryApi.portionFromBulk).mockResolvedValue({
      bulk_out: { id: 'mv-out', tenant_id: 't1', sku_id: 'sku-bulk-1', warehouse_id: 'wh-1', batch_id: null, type: 'split_out', quantity_delta: -412, reason: '', occurred_at: '', created_at: '' },
      unit_in: { id: 'mv-in', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: null, type: 'split_in', quantity_delta: 1, reason: '', occurred_at: '', created_at: '' },
    })

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /Add item from weight pool/i }))
    const dialog = await screen.findByText('Add Item From Weight Pool')
    const modal = dialog.closest('div')!.parentElement!
    await user.type(within(modal).getByLabelText(/actual weight/i), '412')
    await user.click(within(modal).getByRole('button', { name: /^Add item$/ }))

    await waitFor(() => {
      expect(inventoryApi.portionFromBulk).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
        warehouse_id: 'wh-1', weight_grams: 412,
      }))
    })
  })

  it('opens the movement history page for a SKU and can navigate back', async () => {
    const movement: StockMovement = {
      id: 'mv-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-1',
      type: 'receipt', quantity_delta: 22, reason: 'woocommerce import', occurred_at: '2026-08-03T17:07:18Z', created_at: '2026-08-03T17:07:18Z',
    }
    vi.mocked(inventoryApi.listStockMovements).mockResolvedValue([movement])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('WIDGET-1')

    await user.click(screen.getByRole('button', { name: /^History$/ }))

    expect(await screen.findByText('Movement History')).toBeInTheDocument()
    expect(inventoryApi.listStockMovements).toHaveBeenCalledWith('test-token', 'sku-1')
    expect(await screen.findByText('Receipt')).toBeInTheDocument()
    expect(screen.getByText('LOT-1')).toBeInTheDocument()
    expect(screen.getByText('+22')).toBeInTheDocument()
    expect(screen.getByText('woocommerce import')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Inventory/ }))
    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.queryByText('Movement History')).not.toBeInTheDocument()
  })
})

describe('Product-level movement history (account book across all SKUs)', () => {
  it('clicking History on the collapsed product row opens a page without expanding it', async () => {
    vi.mocked(inventoryApi.listStockMovements).mockResolvedValue([])

    const user = userEvent.setup()
    render(<Inventory />)
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'View full movement history' }))

    expect(await screen.findByText('Movement History')).toBeInTheDocument()
    expect(screen.queryByText('WIDGET-1')).not.toBeInTheDocument() // row was never expanded
  })

  it('merges movements across every SKU of the product, sorted newest first, with a running balance', async () => {
    const twoSKUDetail: ProductDetail = {
      product,
      variants: [
        { id: 'var-1', tenant_id: 't1', product_id: 'prod-1', name: 'Small', position: 0, created_at: '', updated_at: '', skus: [sku] },
        {
          id: 'var-2', tenant_id: 't1', product_id: 'prod-1', name: 'Large', position: 1, created_at: '', updated_at: '',
          skus: [{ ...sku, id: 'sku-2', sku_code: 'WIDGET-2' }],
        },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(twoSKUDetail)
    vi.mocked(inventoryApi.listStockMovements).mockImplementation(async (_token, skuId) => {
      if (skuId === 'sku-1') {
        return [
          { id: 'm1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', type: 'receipt', quantity_delta: 10, reason: '', occurred_at: '2026-08-01T00:00:00Z', created_at: '' },
        ]
      }
      return [
        { id: 'm2', tenant_id: 't1', sku_id: 'sku-2', warehouse_id: 'wh-1', type: 'sale', quantity_delta: -3, reason: '', occurred_at: '2026-08-02T00:00:00Z', created_at: '' },
      ]
    })

    const user = userEvent.setup()
    render(<Inventory />)
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'View full movement history' }))
    await screen.findByText('Movement History')

    const rows = (await screen.findAllByRole('row')).slice(1) // drop header row
    expect(rows).toHaveLength(2)
    // Newest first: the sku-2 sale (Aug 2) comes before the sku-1 receipt (Aug 1).
    expect(rows[0].textContent).toContain('WIDGET-2')
    expect(rows[0].textContent).toContain('-3')
    expect(rows[0].textContent).toContain('7') // running balance: 10 - 3
    expect(rows[1].textContent).toContain('WIDGET-1')
    expect(rows[1].textContent).toContain('+10')
    expect(rows[1].textContent).toContain('10') // running balance after the first (oldest) movement
  })

  it('shows an empty state when the product has no movements', async () => {
    vi.mocked(inventoryApi.listStockMovements).mockResolvedValue([])

    const user = userEvent.setup()
    render(<Inventory />)
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'View full movement history' }))

    expect(await screen.findByText('No stock movements recorded for this product yet.')).toBeInTheDocument()
  })

  it('Back returns to the product list', async () => {
    vi.mocked(inventoryApi.listStockMovements).mockResolvedValue([])

    const user = userEvent.setup()
    render(<Inventory />)
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'View full movement history' }))
    await screen.findByText('Movement History')

    await user.click(screen.getByRole('button', { name: /Inventory/ }))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.queryByText('Movement History')).not.toBeInTheDocument()
  })
})

describe('Per-batch movement history', () => {
  it('clicking the history icon on a batch row opens a page scoped to just that batch', async () => {
    const movement: StockMovement = {
      id: 'movement-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-1',
      type: 'receipt', quantity_delta: 6, reason: 'restock', occurred_at: '2026-08-01T00:00:00Z', created_at: '',
    }
    vi.mocked(inventoryApi.listBatchMovements).mockResolvedValue([movement])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('LOT-1')

    await user.click(screen.getByRole('button', { name: /View movement history for batch LOT-1/ }))

    expect(await screen.findByText('Batch Movement History')).toBeInTheDocument()
    expect(inventoryApi.listBatchMovements).toHaveBeenCalledWith('test-token', 'batch-1')
    expect(screen.getByText('Receipt')).toBeInTheDocument()
    expect(screen.getByText('+6')).toBeInTheDocument()
    expect(screen.getByText('restock')).toBeInTheDocument()
    // Not the SKU-wide history endpoint -- this is the batch-scoped one.
    expect(inventoryApi.listStockMovements).not.toHaveBeenCalled()
  })

  it('does not show a history button on the general ledger row', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([
      { ...batch, id: '', lot_number: '', is_general_ledger: true, quantity: 0 },
    ])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('General ledger')

    expect(screen.queryByLabelText(/View movement history for batch/)).not.toBeInTheDocument()
  })

  it('Back returns to the product list', async () => {
    vi.mocked(inventoryApi.listBatchMovements).mockResolvedValue([])

    const user = userEvent.setup()
    render(<Inventory />)
    await user.click(await screen.findByText('Widget'))
    await screen.findByText('LOT-1')

    await user.click(screen.getByRole('button', { name: /View movement history for batch LOT-1/ }))
    await screen.findByText('Batch Movement History')

    await user.click(screen.getByRole('button', { name: /Inventory/ }))
    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.queryByText('Batch Movement History')).not.toBeInTheDocument()
  })
})

describe('Expired batch review queue', () => {
  const expiredBatch: ExpiredBatchReview = {
    batch_id: 'batch-2', sku_id: 'sku-1', sku_code: 'WIDGET-1', product_name: 'Widget',
    warehouse_id: 'wh-1', warehouse_name: 'Main DC', lot_number: 'LOT-2',
    expiry_date: '2026-07-01T00:00:00Z', quantity: 4,
  }

  it('shows no entry button when nothing is pending review', async () => {
    render(<Inventory />)
    await screen.findByText('Widget')

    expect(screen.queryByText(/Expired batches/)).not.toBeInTheDocument()
  })

  it('the Damaged / Lost / Discarded entry is always reachable, even with nothing pending review, and opens straight to the Discarded tab', async () => {
    vi.mocked(inventoryApi.listDiscards).mockResolvedValue({ discards: [], total: 0 })

    render(<Inventory />)
    await screen.findByText('Widget')
    expect(screen.queryByText(/Expired batches/)).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Damaged \/ Lost \/ Discarded/ }))

    expect(await screen.findByText('Expiry & Discards')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Discarded' })).toHaveClass('bg-blue-600')
  })

  it('shows a badge button with the count when batches are pending review, opens the queue on click', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([expiredBatch])

    render(<Inventory />)
    const badge = await screen.findByRole('button', { name: /Expired batches \(1\)/ })
    await userEvent.click(badge)

    expect(await screen.findByText('Expiry & Discards')).toBeInTheDocument()
    expect(screen.getByText('WIDGET-1')).toBeInTheDocument()
    expect(screen.getByText('LOT-2')).toBeInTheDocument()
  })

  it('Discard opens a modal, Confirm calls the API, and removes the batch from the queue', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches)
      .mockResolvedValueOnce([expiredBatch])
      .mockResolvedValue([])
    vi.mocked(inventoryApi.discardBatch).mockResolvedValue({
      id: 'discard-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-2',
      movement_id: 'move-1', type: 'lost', quantity: 4, reason: 'expired', discarded_by: '',
      discarded_at: '2026-08-17T00:00:00Z', created_at: '2026-08-17T00:00:00Z',
    })

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await screen.findByText('LOT-2')

    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))
    await userEvent.type(screen.getByLabelText('Reason'), 'expired')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.discardBatch).toHaveBeenCalledWith('test-token', 'sku-1', {
      warehouse_id: 'wh-1', batch_id: 'batch-2', quantity: 4, reason: 'expired', photo: undefined,
    }))
    expect(await screen.findByText('No expired batches pending review.')).toBeInTheDocument()
  })

  it('Discard modal Cancel closes without calling the API', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([expiredBatch])

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await screen.findByText('LOT-2')

    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByLabelText('Reason')).not.toBeInTheDocument()
    expect(inventoryApi.discardBatch).not.toHaveBeenCalled()
  })

  it('Extend opens a date field, Confirm calls the API with the chosen date', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches)
      .mockResolvedValueOnce([expiredBatch])
      .mockResolvedValue([])
    vi.mocked(inventoryApi.extendBatchExpiry).mockResolvedValue(null)

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await screen.findByText('LOT-2')

    await userEvent.click(screen.getByRole('button', { name: 'Extend' }))
    await userEvent.type(screen.getByLabelText('New expiry date'), '2026-12-01')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.extendBatchExpiry).toHaveBeenCalledWith('test-token', 'batch-2', { new_expiry_date: '2026-12-01' }))
  })

  it('Dismiss calls the API and removes the batch from the queue without a confirm prompt', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches)
      .mockResolvedValueOnce([expiredBatch])
      .mockResolvedValue([])
    vi.mocked(inventoryApi.dismissExpiredBatch).mockResolvedValue(null)
    const confirmSpy = vi.spyOn(window, 'confirm')

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await screen.findByText('LOT-2')

    await userEvent.click(screen.getByRole('button', { name: 'Dismiss' }))

    await waitFor(() => expect(inventoryApi.dismissExpiredBatch).toHaveBeenCalledWith('test-token', 'batch-2'))
    expect(confirmSpy).not.toHaveBeenCalled()
  })

  it('shows an error if discard fails', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([expiredBatch])
    vi.mocked(inventoryApi.discardBatch).mockRejectedValue(new Error('insufficient stock'))

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await screen.findByText('LOT-2')

    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(await screen.findByText('insufficient stock')).toBeInTheDocument()
  })

  it('Back returns to the product list', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([expiredBatch])

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await screen.findByText('Expiry & Discards')

    await userEvent.click(screen.getByRole('button', { name: /Inventory/ }))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
  })

  it('Expiring soon tab lists batches not yet expired and can discard one', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([expiredBatch])
    vi.mocked(inventoryApi.listExpiringBatches).mockResolvedValue([{
      batch_id: 'batch-3', sku_id: 'sku-2', sku_code: 'GADGET-1', product_name: 'Gadget',
      warehouse_id: 'wh-1', warehouse_name: 'Main DC', lot_number: 'LOT-3',
      expiry_date: '2026-09-01T00:00:00Z', quantity: 7,
    }])
    vi.mocked(inventoryApi.discardBatch).mockResolvedValue({
      id: 'discard-2', tenant_id: 't1', sku_id: 'sku-2', warehouse_id: 'wh-1', batch_id: 'batch-3',
      movement_id: 'move-2', type: 'lost', quantity: 7, reason: '', discarded_by: '',
      discarded_at: '2026-08-17T00:00:00Z', created_at: '2026-08-17T00:00:00Z',
    })

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Expiring soon' }))

    expect(await screen.findByText('GADGET-1')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(inventoryApi.discardBatch).toHaveBeenCalledWith('test-token', 'sku-2', {
      warehouse_id: 'wh-1', batch_id: 'batch-3', quantity: 7, reason: '', photo: undefined,
    }))
  })

  it('Discarded tab shows the report, including a photo thumbnail when present', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([expiredBatch])
    vi.mocked(inventoryApi.listDiscards).mockResolvedValue({
      discards: [{
        id: 'discard-1', tenant_id: 't1', sku_id: 'sku-2', sku_code: 'GADGET-1', product_name: 'Gadget',
        warehouse_id: 'wh-1', warehouse_name: 'Main DC', batch_id: 'batch-3', lot_number: 'LOT-3',
        movement_id: 'move-1', type: 'lost', quantity: 3, reason: 'expired', photo_url: 'https://example.com/photo.jpg',
        discarded_by: 'user-1', discarded_at: '2026-08-15T00:00:00Z', created_at: '2026-08-15T00:00:00Z',
      }],
      total: 1,
    })

    render(<Inventory />)
    await userEvent.click(await screen.findByRole('button', { name: /Expired batches \(1\)/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Discarded' }))

    expect(await screen.findByText('GADGET-1')).toBeInTheDocument()
    expect(screen.getByText('Gadget')).toBeInTheDocument()
    expect(screen.getByText('expired')).toBeInTheDocument()
    expect(screen.getByRole('img')).toHaveAttribute('src', 'https://example.com/photo.jpg')
  })
})
