// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import { act } from 'react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import i18next from 'i18next'
import Procurement from '../Procurement'
import { renderAt } from '../../test/renderAt'
import type { PurchaseOrder, PurchaseOrderDetail, PurchaseOrderLineItem } from '../../procurement/types'
import type { Supplier } from '../../suppliers/types'
import type { Warehouse } from '../../inventory/types'
import type { Product, ProductDetail } from '../../pms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../procurement/api', () => ({
  listPurchaseOrders: vi.fn(),
  createPurchaseOrder: vi.fn(),
  getPurchaseOrder: vi.fn(),
  updatePurchaseOrder: vi.fn(),
  deletePurchaseOrder: vi.fn(),
  addLineItem: vi.fn(),
  updateLineItem: vi.fn(),
  removeLineItem: vi.fn(),
  attachDocumentToPurchaseOrder: vi.fn(),
  deletePurchaseOrderDocument: vi.fn(),
  getDocumentParsingJob: vi.fn(),
  retryDocumentParsingJob: vi.fn(),
  createPurchaseOrderFromDocument: vi.fn(),
  linkLineItemToSku: vi.fn(),
  createProductAndLinkLineItem: vi.fn(),
  createSupplierAndAssignToPurchaseOrder: vi.fn(),
  uploadInvoice: vi.fn(),
  getReceivingParsingJob: vi.fn(),
  listReceivingRecords: vi.fn(),
  getReceivingRecord: vi.fn(),
  completeReceiving: vi.fn(),
  acknowledgeReceivingDiscrepancy: vi.fn(),
  linkReceivingDiscrepancies: vi.fn(),
  flagDamagedLineItem: vi.fn(),
  listBookingDiscrepancies: vi.fn(),
}))

vi.mock('../../suppliers/api', () => ({
  listSuppliers: vi.fn(),
  createSupplier: vi.fn(),
  updateSupplier: vi.fn(),
  deleteSupplier: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  createWarehouse: vi.fn(),
  updateWarehouse: vi.fn(),
  deleteWarehouse: vi.fn(),
  listBatches: vi.fn(),
  createBatch: vi.fn(),
  updateBatch: vi.fn(),
  deleteBatch: vi.fn(),
  getSKUStockLevel: vi.fn(),
  recordStockMovement: vi.fn(),
  listStockMovements: vi.fn(),
  recordRecount: vi.fn(),
  transferStock: vi.fn(),
  markDamagedOrLost: vi.fn(),
  listExpiringBatches: vi.fn(),
  receiveStock: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
  updateTenantSettings: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  createProduct: vi.fn(),
  updateProduct: vi.fn(),
  deleteProduct: vi.fn(),
}))

vi.mock('../../integrations/api', () => ({
  listAIConnections: vi.fn(),
}))

import * as procurementApi from '../../procurement/api'
import * as suppliersApi from '../../suppliers/api'
import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../../pms/api'
import * as settingsApi from '../../settings/api'
import * as integrationsApi from '../../integrations/api'
import { acceptConfirm, dismissConfirm } from '../../test/confirm'
import { formatMoney } from '../../format/money'

function po(overrides: Partial<PurchaseOrder> = {}): PurchaseOrder {
  return {
    id: 'po-1', tenant_id: 'tenant-1', supplier_id: 'supplier-1', warehouse_id: 'warehouse-1',
    status: 'draft', currency_code: 'USD', expected_date: null, notes: '',
    is_potential_duplicate: false, total_minor_units: 0,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function supplier(overrides: Partial<Supplier> = {}): Supplier {
  return {
    id: 'supplier-1', tenant_id: 'tenant-1', name: 'Acme Foods',
    email: '', phone: '', country: '', lead_time_days: null, status: 'active', notes: '',
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function warehouse(overrides: Partial<Warehouse> = {}): Warehouse {
  return {
    id: 'warehouse-1', tenant_id: 'tenant-1', name: 'Main Warehouse',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function lineItem(overrides: Partial<PurchaseOrderLineItem> = {}): PurchaseOrderLineItem {
  return {
    id: 'line-item-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', sku_id: null, raw_sku: '',
    raw_description: 'Widget A', position: 0, quantity: 10, units_per_purchased_item: 1, unit_cost_minor_units: 500,
    line_type: 'product', line_type_confirmed: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function detail(overrides: Partial<PurchaseOrderDetail> = {}): PurchaseOrderDetail {
  return { purchase_order: po(), line_items: [], documents: [], ...overrides }
}

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Widget Product',
    slug: 'widget-product', description: '', status: 'active', has_variants: false, tags: [],
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function receivingRecord(overrides: Partial<import('../../procurement/types').ReceivingRecord> = {}): import('../../procurement/types').ReceivingRecord {
  return {
    id: 'receiving-record-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1',
    invoice_file_url: 'https://example.com/invoice.csv', invoice_file_name: 'invoice.csv', content_type: 'text/csv',
    status: 'pending_review', is_potential_duplicate: false, created_at: '', updated_at: '',
    ...overrides,
  }
}

function receivingDetail(overrides: Partial<import('../../procurement/types').ReceivingRecordDetail> = {}): import('../../procurement/types').ReceivingRecordDetail {
  return { receiving_record: receivingRecord(), discrepancies: [], ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
  vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([])
  vi.mocked(procurementApi.listBookingDiscrepancies).mockResolvedValue({ missing_items: [], damaged_lost: [] })
  vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
  // Empty default_currency means the "no tenant setting yet" fallback (USD)
  // applies -- individual tests override this to verify the real default.
  vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  // Configured by default so existing PDF-upload tests aren't interrupted
  // by the "no AI configured" confirm prompt -- tests for that prompt
  // specifically override this to null.
  vi.mocked(integrationsApi.listAIConnections).mockResolvedValue([{
    id: 'ai-conn-1', position: 0, label: '', enabled: true, has_api_key: true, provider: 'openai', status: 'connected', created_at: '',
  }])
})

describe('Purchase orders list', () => {
  it('shows an empty state when there are no purchase orders', async () => {
    renderAt(<Procurement />, '/procurement')
    expect(await screen.findByText('No purchase orders yet')).toBeInTheDocument()
  })

  it('renders its texts in pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    try {
      renderAt(<Procurement />, '/procurement')
      expect(await screen.findByText('Nenhum pedido de compra ainda')).toBeInTheDocument()
    } finally {
      await i18next.changeLanguage('en')
    }
  })

  it('lists purchase orders resolving supplier and warehouse names', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])

    renderAt(<Procurement />, '/procurement')

    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
    expect(screen.getByText('Main Warehouse')).toBeInTheDocument()
    expect(screen.getByText('draft', { selector: 'span' })).toBeInTheDocument()
  })

  it('shows each purchase order total on the main list', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ total_minor_units: 13000, currency_code: 'EUR' })])

    renderAt(<Procurement />, '/procurement')

    expect(await screen.findByText(formatMoney(13000, 'EUR'))).toBeInTheDocument()
  })

  it('filters by status tab', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([
      po({ id: 'po-1', status: 'draft' }),
      po({ id: 'po-2', status: 'sent', supplier_id: 'supplier-1' }),
    ])

    renderAt(<Procurement />, '/procurement')
    await screen.findAllByText('Acme Foods')
    expect(screen.getByText((_, node) => node?.tagName === 'SPAN' && node.textContent === '2 purchase orders')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'sent' }))

    expect(await screen.findByText((_, node) => node?.tagName === 'SPAN' && node.textContent === '1 purchase order')).toBeInTheDocument()
  })

  it('deletes a purchase order after confirming', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(procurementApi.deletePurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1'))
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByTitle('Delete'))

    await dismissConfirm()
    expect(procurementApi.deletePurchaseOrder).not.toHaveBeenCalled()
  })

  it('shows a duplicate-order badge and opens the other PO when clicked', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([
      po({ id: 'po-2', is_potential_duplicate: true, duplicate_of_purchase_order_id: 'po-1' }),
    ])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-1' }) }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Possible duplicate order'))

    await waitFor(() => expect(procurementApi.getPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1'))
  })

  it('does not show a duplicate-order badge for a normal PO', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])

    renderAt(<Procurement />, '/procurement')
    await screen.findByText('Acme Foods')

    expect(screen.queryByText('Possible duplicate order')).not.toBeInTheDocument()
  })
})

describe('Purchase order create', () => {
  it('opening Add Purchase Order shows a blank create form with no line items section', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Add purchase order' }))

    expect(await screen.findByText('New purchase order', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create purchase order' })).toBeInTheDocument()
    expect(screen.queryByText('Line items')).not.toBeInTheDocument()
  })

  it('creates a new purchase order and returns to the list', async () => {
    vi.mocked(procurementApi.createPurchaseOrder).mockResolvedValue(po())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail())

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Add purchase order' }))
    await userEvent.selectOptions(await screen.findByLabelText('Supplier'), 'supplier-1')
    await userEvent.selectOptions(screen.getByLabelText('Warehouse (optional)'), 'warehouse-1')
    await userEvent.click(screen.getByRole('button', { name: 'Create purchase order' }))

    await waitFor(() => expect(procurementApi.createPurchaseOrder).toHaveBeenCalledWith('test-token', expect.objectContaining({
      supplier_id: 'supplier-1', warehouse_id: 'warehouse-1', currency_code: 'USD',
      expected_date: new Date().toISOString().slice(0, 10),
    })))
  })

  it('defaults currency to a select with USD chosen and expected date to today', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Add purchase order' }))

    expect(screen.getByLabelText('Currency').tagName).toBe('SELECT')
    expect(screen.getByLabelText('Currency')).toHaveValue('USD')
    expect(screen.getByLabelText('Expected date')).toHaveValue(new Date().toISOString().slice(0, 10))
  })

  it('defaults currency to the tenant system-settings default once it loads', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Add purchase order' }))

    await waitFor(() => expect(screen.getByLabelText('Currency')).toHaveValue('EUR'))
  })

  it('does not override a currency the user already picked once settings load', async () => {
    let resolveSettings: (v: import('../../settings/types').TenantSettings) => void
    vi.mocked(settingsApi.getTenantSettings).mockReturnValue(new Promise(resolve => { resolveSettings = resolve }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Add purchase order' }))
    await userEvent.selectOptions(await screen.findByLabelText('Currency'), 'GBP')

    resolveSettings!({ tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '', prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '' })

    // Give the settings promise's .then() a tick to run before asserting —
    // still wrapped in waitFor so the resulting state update is act()-safe.
    await waitFor(() => expect(screen.getByLabelText('Currency')).toHaveValue('GBP'))
  })
})

describe('Upload document to create a PO', () => {
  it('opens a modal with just a file field -- no supplier, warehouse, currency, or date to fill in', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Upload document' }))

    expect(await screen.findByText('Upload document', { selector: 'h2' })).toBeInTheDocument()
    expect(screen.getByLabelText('Document')).toBeInTheDocument()
    expect(screen.queryByLabelText(/Supplier/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Warehouse/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Currency/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/date/i)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Upload & create' })).toBeDisabled()
  })

  it('navigates to the newly created PO immediately, without waiting for the background parse', async () => {
    vi.mocked(procurementApi.createPurchaseOrderFromDocument).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-new', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-new' }) }))
    // Never resolves within the test -- if navigation were still blocking
    // on this, the test would time out waiting for the PO page instead of
    // finding it right away.
    vi.mocked(procurementApi.getDocumentParsingJob).mockImplementation(() => new Promise(() => {}))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Upload document' }))
    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Document'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload & create' }))

    await waitFor(() => expect(procurementApi.createPurchaseOrderFromDocument).toHaveBeenCalledWith('test-token', {
      currency_code: 'USD', file,
    }))
    expect(await screen.findByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('shows a parsing indicator on the new PO while the background parse is still running', async () => {
    vi.mocked(procurementApi.createPurchaseOrderFromDocument).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-new', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-new' }) }))
    vi.mocked(procurementApi.getDocumentParsingJob).mockImplementation(() => new Promise(() => {}))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Upload document' }))
    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Document'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload & create' }))

    expect(await screen.findByText('Parsing document…')).toBeInTheDocument()
    // polling starts in an effect right after the spinner renders
    await waitFor(() => expect(procurementApi.getDocumentParsingJob).toHaveBeenCalledWith('test-token', 'job-1'))
  })

  it('picks up line items on the new PO once the background parse finishes', async () => {
    vi.mocked(procurementApi.createPurchaseOrderFromDocument).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-new', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-new', document_id: 'document-1',
      status: 'completed', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ id: 'po-new' }), line_items: [lineItem()],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Upload document' }))
    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Document'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload & create' }))

    // No second upload needed to see what the first one extracted -- the
    // PO page itself picks it up once the background parse completes.
    expect(await screen.findByText('Widget A')).toBeInTheDocument()
  })

  it('shows an error if the upload fails', async () => {
    vi.mocked(procurementApi.createPurchaseOrderFromDocument).mockRejectedValue(new Error('document is missing required columns'))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Upload document' }))
    await userEvent.upload(await screen.findByLabelText('Document'), new File(['bad'], 'bad.csv', { type: 'text/csv' }))
    await userEvent.click(screen.getByRole('button', { name: 'Upload & create' }))

    expect(await screen.findByText('document is missing required columns')).toBeInTheDocument()
  })
})

describe('Purchase order edit with line items', () => {
  beforeEach(() => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ line_items: [lineItem()] }))
  })

  it('prefills header fields and shows existing line items', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('10')).toBeInTheDocument()
    expect(screen.getByLabelText('Supplier')).toHaveValue('supplier-1')
  })

  it('adds a new line item', async () => {
    vi.mocked(procurementApi.addLineItem).mockResolvedValue(lineItem({ id: 'line-item-2', raw_description: 'Widget B', quantity: 5 }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    await userEvent.click(screen.getAllByRole('button', { name: /Add line item/ })[0])
    await userEvent.type(await screen.findByLabelText('Line item description'), 'Widget B')
    await userEvent.clear(screen.getByLabelText('Quantity'))
    await userEvent.type(screen.getByLabelText('Quantity'), '5')
    await userEvent.clear(screen.getByLabelText('Unit cost'))
    await userEvent.type(screen.getByLabelText('Unit cost'), '12.50')
    await userEvent.click(screen.getByRole('button', { name: 'Save line' }))

    await waitFor(() => expect(procurementApi.addLineItem).toHaveBeenCalledWith('test-token', 'po-1', {
      sku_id: null, raw_description: 'Widget B', quantity: 5, units_per_purchased_item: 1, unit_cost_minor_units: 1250, line_type: 'product',
    }))
  })

  it('adds a line item from the button at the end of the list, not just the toolbar one', async () => {
    vi.mocked(procurementApi.addLineItem).mockResolvedValue(lineItem({ id: 'line-item-2', raw_description: 'Widget B', quantity: 5 }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const addButtons = screen.getAllByRole('button', { name: /Add line item/ })
    expect(addButtons).toHaveLength(2)
    await userEvent.click(addButtons[1])
    await userEvent.type(await screen.findByLabelText('Line item description'), 'Widget B')
    await userEvent.click(screen.getByRole('button', { name: 'Save line' }))

    await waitFor(() => expect(procurementApi.addLineItem).toHaveBeenCalledWith('test-token', 'po-1', expect.objectContaining({
      raw_description: 'Widget B',
    })))
  })

  it('adds a shipping charge with no SKU picker, since it is not a product', async () => {
    vi.mocked(procurementApi.addLineItem).mockResolvedValue(lineItem({ id: 'line-item-2', raw_description: 'Shipping', quantity: 1, line_type: 'shipping' }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    await userEvent.click(screen.getAllByRole('button', { name: /Add line item/ })[0])
    await userEvent.type(await screen.findByLabelText('Line item description'), 'Shipping')
    await userEvent.selectOptions(screen.getByLabelText('Line item type'), 'shipping')
    expect(screen.queryByRole('button', { name: /Link product/ })).not.toBeInTheDocument()
    await userEvent.clear(screen.getByLabelText('Unit cost'))
    await userEvent.type(screen.getByLabelText('Unit cost'), '15.00')
    await userEvent.click(screen.getByRole('button', { name: 'Save line' }))

    await waitFor(() => expect(procurementApi.addLineItem).toHaveBeenCalledWith('test-token', 'po-1', {
      sku_id: null, raw_description: 'Shipping', quantity: 1, units_per_purchased_item: 1, unit_cost_minor_units: 1500, line_type: 'shipping',
    }))
  })

  it('shows a grand total row summing every line item, discounts subtracting', async () => {
    // unit_cost_minor_units is always stored non-negative -- a $10.00
    // discount is entered as 1000, not -1000, and the total calc itself
    // must flip the sign for line_type 'discount'/'credit_note'.
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [
        lineItem({ id: 'line-item-1', quantity: 10, unit_cost_minor_units: 500 }), // 5000
        lineItem({ id: 'line-item-2', raw_description: 'Shipping', sku_id: null, quantity: 1, unit_cost_minor_units: 1500, line_type: 'shipping', line_type_confirmed: true }), // +1500
        lineItem({ id: 'line-item-3', raw_description: 'Discount', sku_id: null, quantity: 1, unit_cost_minor_units: 1000, line_type: 'discount', line_type_confirmed: true }), // -1000
      ],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    expect(await screen.findByText(formatMoney(5500, 'USD'))).toBeInTheDocument()
    expect(await screen.findByText(formatMoney(-1000, 'USD'))).toBeInTheDocument()
  })

  it('shows product name and SKU code, not the raw sku_id, when editing an already-linked line item', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ sku_id: 'sku-1', product_name: 'Acme Widget', sku_code: 'WID-001' })],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText('Acme Widget — WID-001')).toBeInTheDocument()
    expect(screen.queryByText('sku-1')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Units per purchased item')).toBeInTheDocument()
  })

  it('shows a charge row without a SKU picker or a needs-review flag', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem(), lineItem({ id: 'line-item-2', raw_description: 'Shipping', sku_id: null, line_type: 'shipping', line_type_confirmed: true })],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    const shippingLabels = await screen.findAllByText('Shipping')
    const row = shippingLabels[0].closest('tr')!
    expect(within(row).getAllByText('Shipping').length).toBe(2)
    expect(within(row).queryByText('Needs review')).not.toBeInTheDocument()
    // Two "—" placeholders: Linked SKU (charge rows never need one) and
    // Batch (no suggestion on this fixture).
    expect(within(row).getAllByText('—').length).toBe(2)
  })

  it('shows a document-parsed lot number and expiry date suggestion on a line item', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ suggested_lot_number: 'PT26004', suggested_expiry_date: '2027-03-01' })],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('PT26004')).toBeInTheDocument()
    expect(screen.getByText(/exp 2027-03-01/)).toBeInTheDocument()
  })

  it('shows a placeholder for a line item with no batch suggestion', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ line_items: [lineItem()] }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    const row = (await screen.findByText('Widget A')).closest('tr')!
    expect(within(row).getByText('—')).toBeInTheDocument()
  })

  it('shows a linked line item as product name and SKU code, not the raw SKU id', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ sku_id: 'sku-1', product_name: 'Acme Widget', sku_code: 'WID-001' })],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Acme Widget')).toBeInTheDocument()
    expect(screen.getByText('— WID-001')).toBeInTheDocument()
    expect(screen.queryByText('sku-1')).not.toBeInTheDocument()
  })

  it('falls back to the raw SKU id when a linked item has no resolved product name', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ sku_id: 'sku-orphaned', product_name: null, sku_code: null })],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('sku-orphaned')).toBeInTheDocument()
  })

  it('shows a Confirm action for an unconfirmed AI-classified charge row, and saves on click', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ id: 'line-item-2', raw_description: 'Shipping', sku_id: null, unit_cost_minor_units: 1500, line_type: 'shipping', line_type_confirmed: false })],
    }))
    vi.mocked(procurementApi.updateLineItem).mockResolvedValue(lineItem({
      id: 'line-item-2', raw_description: 'Shipping', line_type: 'shipping', line_type_confirmed: true,
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    await userEvent.click(await screen.findByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(procurementApi.updateLineItem).toHaveBeenCalledWith('test-token', 'line-item-2', {
      sku_id: null, raw_sku: '', raw_description: 'Shipping', quantity: 10, unit_cost_minor_units: 1500, line_type: 'shipping',
    }))
  })

  it('removes a line item after confirming', async () => {

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    const row = (await screen.findByText('Widget A')).closest('tr')!
    await userEvent.click(within(row).getByTitle('Remove'))

    await acceptConfirm()
    await waitFor(() => expect(procurementApi.removeLineItem).toHaveBeenCalledWith('test-token', 'line-item-1'))
  })

  it('links a line item to a product SKU via the picker', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product()], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [{
        id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0,
        created_at: '', updated_at: '',
        skus: [{
          id: 'sku-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: 'WID-001',
          price_minor_units: 999, currency_code: 'USD', is_virtual: false, status: 'active',
          created_at: '', updated_at: '',
        }],
      }],
    } satisfies ProductDetail)
    vi.mocked(procurementApi.addLineItem).mockResolvedValue(lineItem({ id: 'line-item-2', sku_id: 'sku-1' }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    await userEvent.click(screen.getAllByRole('button', { name: /Add line item/ })[0])
    await userEvent.click(await screen.findByRole('button', { name: /Link product/ }))
    await userEvent.type(await screen.findByPlaceholderText('Search products by name…'), 'Widget')
    await userEvent.click(await screen.findByText('Widget Product'))
    await userEvent.click(await screen.findByText('WID-001'))

    expect(screen.queryByPlaceholderText('Search products by name…')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Save line' }))

    await waitFor(() => expect(procurementApi.addLineItem).toHaveBeenCalledWith('test-token', 'po-1', expect.objectContaining({
      sku_id: 'sku-1',
    })))
  })

  it('Discard returns to the list without saving', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(procurementApi.updatePurchaseOrder).not.toHaveBeenCalled()
    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
  })

  it('re-fetches the list on returning to it, so a status change made on the edit page is not left stale', async () => {
    // First list load shows the PO as draft -- after editing status to
    // 'sent' and going back, the list must be re-fetched (not read from
    // whatever it loaded on first mount) to show the new status.
    vi.mocked(procurementApi.listPurchaseOrders)
      .mockResolvedValueOnce([po({ status: 'draft' })])
      .mockResolvedValueOnce([po({ status: 'sent' })])
    vi.mocked(procurementApi.updatePurchaseOrder).mockResolvedValue(po({ status: 'sent' }))

    renderAt(<Procurement />, '/procurement')
    const draftRow = (await screen.findByText('Acme Foods')).closest('tr')!
    expect(within(draftRow).getByText('draft')).toBeInTheDocument()

    await userEvent.click(screen.getByText('Acme Foods'))
    await screen.findByText('Widget A')
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'sent')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(procurementApi.updatePurchaseOrder).toHaveBeenCalledWith(
      'test-token', 'po-1', expect.objectContaining({ status: 'sent' }),
    ))
    await userEvent.click(screen.getByRole('button', { name: 'Purchase Orders' }))

    expect(procurementApi.listPurchaseOrders).toHaveBeenCalledTimes(2)
    const row = (await screen.findByText('Acme Foods')).closest('tr')!
    expect(within(row).getByText('sent')).toBeInTheDocument()
  })

  it('uploading a document to an existing PO shows a parsing indicator', async () => {
    vi.mocked(procurementApi.attachDocumentToPurchaseOrder).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    // Never resolves within the test -- keeps the job in-flight so the
    // "parsing" indicator stays visible long enough to assert on, same as
    // it would while cmd/scheduler is still working on the real job.
    vi.mocked(procurementApi.getDocumentParsingJob).mockImplementation(() => new Promise(() => {}))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    await waitFor(() => expect(procurementApi.attachDocumentToPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1', file))
    expect(await screen.findByText('Parsing document…')).toBeInTheDocument()
  })

  it('surfaces a friendly message when AI parsing is not configured, pointing at manual entry', async () => {
    vi.mocked(procurementApi.attachDocumentToPurchaseOrder).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'failed', error_message: 'AI document parsing is not configured', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    expect(await screen.findByText(/add line items manually/)).toBeInTheDocument()
    expect(screen.queryByText('AI document parsing is not configured')).not.toBeInTheDocument()
  })

  it('warns before uploading a PDF when no AI provider is configured, and still uploads if confirmed', async () => {
    vi.mocked(integrationsApi.listAIConnections).mockResolvedValue([])
    vi.mocked(procurementApi.attachDocumentToPurchaseOrder).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'completed', error_message: '', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    await acceptConfirm()
    await waitFor(() => expect(procurementApi.attachDocumentToPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1', file))
  })

  it('does not upload a PDF when the no-AI warning is dismissed', async () => {
    vi.mocked(integrationsApi.listAIConnections).mockResolvedValue([])

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    await dismissConfirm()
    expect(procurementApi.attachDocumentToPurchaseOrder).not.toHaveBeenCalled()
  })

  it('does not warn before uploading a CSV, since it never needs AI', async () => {
    vi.mocked(integrationsApi.listAIConnections).mockResolvedValue([])
    vi.mocked(procurementApi.attachDocumentToPurchaseOrder).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'completed', error_message: '', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    await waitFor(() => expect(procurementApi.attachDocumentToPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1', file))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })
})

describe('Documents', () => {
  it('shows a link to each uploaded document', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    const link = await screen.findByText('invoice.pdf')
    expect(link.closest('a')).toHaveAttribute('href', 'https://uploads.example.com/tenant-1/abc.pdf')
  })

  it('does not render a Documents section when there are none', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail())

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByLabelText('Supplier')

    expect(screen.queryByText('Documents')).not.toBeInTheDocument()
  })

  it('shows a "possible duplicate" badge when flagged', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: true, created_at: '',
      }],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Possible duplicate')).toBeInTheDocument()
  })

  it('deletes a document after confirming', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))
    vi.mocked(procurementApi.deletePurchaseOrderDocument).mockResolvedValue(null)

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(procurementApi.deletePurchaseOrderDocument).toHaveBeenCalledWith('test-token', 'document-1'))
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Delete'))

    await dismissConfirm()
    expect(procurementApi.deletePurchaseOrderDocument).not.toHaveBeenCalled()
  })

  it('reparses a document and clears once it completes', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))
    vi.mocked(procurementApi.retryDocumentParsingJob).mockResolvedValue({
      id: 'job-2', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1', status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-2', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1', status: 'completed', error_message: '', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Reparse'))

    await waitFor(() => expect(procurementApi.retryDocumentParsingJob).toHaveBeenCalledWith('test-token', 'document-1'))
    await waitFor(() => expect(procurementApi.getDocumentParsingJob).toHaveBeenCalledWith('test-token', 'job-2'))
    // getPurchaseOrder called again for the initial load + the post-reparse refresh
    await waitFor(() => expect(procurementApi.getPurchaseOrder).toHaveBeenCalledTimes(2))
  })

  it('shows an error inline when the reparse itself fails', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))
    vi.mocked(procurementApi.retryDocumentParsingJob).mockRejectedValue(new Error('purchase order document not found'))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Reparse'))

    expect(await screen.findByText('purchase order document not found')).toBeInTheDocument()
  })

  it('shows the friendly parse-failed message when the retried job fails again', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))
    vi.mocked(procurementApi.retryDocumentParsingJob).mockResolvedValue({
      id: 'job-2', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1', status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-2', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1', status: 'failed', error_message: 'unmarshal intent: unexpected EOF', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Reparse'))

    expect(await screen.findByText('unmarshal intent: unexpected EOF')).toBeInTheDocument()
  })
})

describe('Reviewing unmatched line items', () => {
  beforeEach(() => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ status: 'needs_review' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ status: 'needs_review' }),
      line_items: [lineItem({ id: 'line-item-unmatched', sku_id: null, raw_sku: 'UNK-999', raw_description: 'Mystery Item' })],
    }))
  })

  it('shows a Needs review badge for unmatched line items', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Needs review')).toBeInTheDocument()
  })

  it('links an unmatched item to an existing product via the review picker', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product()], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [{
        id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0,
        created_at: '', updated_at: '',
        skus: [{
          id: 'sku-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: 'WID-001',
          price_minor_units: 999, currency_code: 'USD', is_virtual: false, status: 'active',
          created_at: '', updated_at: '',
        }],
      }],
    } satisfies ProductDetail)
    vi.mocked(procurementApi.linkLineItemToSku).mockResolvedValue(lineItem({ id: 'line-item-unmatched', sku_id: 'sku-1' }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.click(await screen.findByText('Needs review'))
    await userEvent.click(await screen.findByRole('button', { name: /Link to existing product/ }))
    await userEvent.type(await screen.findByPlaceholderText('Search products by name…'), 'Widget')
    await userEvent.click(await screen.findByText('Widget Product'))
    await userEvent.click(await screen.findByText('WID-001'))

    await waitFor(() => expect(procurementApi.linkLineItemToSku).toHaveBeenCalledWith('test-token', 'line-item-unmatched', { sku_id: 'sku-1', units_per_purchased_item: 1 }))
  })

  // "case of a product" edge case: a supplier's box line (e.g.
  // "Rapadurinha 36x25gr") links to the individual unit SKU with an
  // explicit multiplier for how many stock units one purchased box holds.
  it('links an unmatched item with a units-per-purchased-item multiplier for a case/box line', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ id: 'line-item-unmatched', sku_id: null, raw_sku: 'BAR019', raw_description: 'Rapadurinha 36x25gr' })],
    }))
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product({ name: 'Rapadurinha de Minas 25g' })], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product({ name: 'Rapadurinha de Minas 25g' }),
      variants: [{
        id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0,
        created_at: '', updated_at: '',
        skus: [{
          id: 'sku-unit-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: '7896289400005',
          price_minor_units: 100, currency_code: 'USD', is_virtual: false, status: 'active',
          created_at: '', updated_at: '',
        }],
      }],
    } satisfies ProductDetail)
    vi.mocked(procurementApi.linkLineItemToSku).mockResolvedValue(lineItem({ id: 'line-item-unmatched', sku_id: 'sku-unit-1', units_per_purchased_item: 36 }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.click(await screen.findByText('Needs review'))
    await userEvent.click(await screen.findByRole('button', { name: /Link to existing product/ }))
    await userEvent.clear(await screen.findByLabelText('Units per purchased item'))
    await userEvent.type(screen.getByLabelText('Units per purchased item'), '36')
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Rapadurinha')
    await userEvent.click(await screen.findByText('Rapadurinha de Minas 25g'))
    await userEvent.click(await screen.findByText('7896289400005'))

    await waitFor(() => expect(procurementApi.linkLineItemToSku).toHaveBeenCalledWith('test-token', 'line-item-unmatched', {
      sku_id: 'sku-unit-1', units_per_purchased_item: 36,
    }))
  })

  it('creates a new product and links an unmatched item', async () => {
    vi.mocked(procurementApi.createProductAndLinkLineItem).mockResolvedValue(lineItem({ id: 'line-item-unmatched', sku_id: 'new-sku-1' }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.click(await screen.findByText('Needs review'))
    await userEvent.click(await screen.findByRole('button', { name: /Create new product/ }))

    const nameInput = await screen.findByDisplayValue('Mystery Item')
    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, 'Mystery Widget')
    await userEvent.click(screen.getByRole('button', { name: 'Create & link' }))

    await waitFor(() => expect(procurementApi.createProductAndLinkLineItem).toHaveBeenCalledWith('test-token', 'line-item-unmatched', expect.objectContaining({
      product_name: 'Mystery Widget', sku_code: 'UNK-999',
    })))
  })
})

describe('Missing supplier review', () => {
  beforeEach(() => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ supplier_id: null, status: 'needs_review' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ supplier_id: null, status: 'needs_review' }),
      line_items: [],
    }))
  })

  it('shows Missing supplier in the list and a review banner on the edit page', async () => {
    renderAt(<Procurement />, '/procurement')

    expect(await screen.findByText('Missing supplier')).toBeInTheDocument()

    await userEvent.click(screen.getByText('Missing supplier'))

    expect(await screen.findByText(/this purchase order has no supplier/)).toBeInTheDocument()
  })

  it('creates a supplier and assigns it in one click', async () => {
    vi.mocked(procurementApi.createSupplierAndAssignToPurchaseOrder).mockResolvedValue(
      po({ supplier_id: 'new-supplier-1', status: 'draft' }),
    )

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Missing supplier'))
    await userEvent.type(await screen.findByLabelText('New supplier name'), 'Brand New Supplier')
    await userEvent.click(screen.getByRole('button', { name: 'Create & assign' }))

    await waitFor(() => expect(procurementApi.createSupplierAndAssignToPurchaseOrder).toHaveBeenCalledWith(
      'test-token', 'po-1', { name: 'Brand New Supplier' },
    ))
  })

  it('pre-fills the supplier name from the document extraction suggestion, still editable', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ supplier_id: null, status: 'needs_review', suggested_supplier_name: 'Pedaços do Brasil, Lda.' }),
      line_items: [],
    }))
    vi.mocked(procurementApi.createSupplierAndAssignToPurchaseOrder).mockResolvedValue(
      po({ supplier_id: 'new-supplier-1', status: 'draft' }),
    )

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Missing supplier'))

    const nameInput = await screen.findByLabelText('New supplier name')
    expect(nameInput).toHaveValue('Pedaços do Brasil, Lda.')

    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, 'Corrected Name')
    await userEvent.click(screen.getByRole('button', { name: 'Create & assign' }))

    await waitFor(() => expect(procurementApi.createSupplierAndAssignToPurchaseOrder).toHaveBeenCalledWith(
      'test-token', 'po-1', { name: 'Corrected Name' },
    ))
  })

  it('pre-selects the matching supplier and collapses the create-new box, when the suggestion matches an existing supplier', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier({ id: 'supplier-9', name: 'Pedaços do Brasil, Lda.' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ supplier_id: null, status: 'needs_review', suggested_supplier_name: 'PEDAÇOS DO BRASIL, LDA.' }),
      line_items: [],
    }))
    vi.mocked(procurementApi.createSupplierAndAssignToPurchaseOrder).mockResolvedValue(
      po({ supplier_id: 'supplier-9', status: 'draft' }),
    )

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Missing supplier'))

    expect(await screen.findByLabelText('Supplier')).toHaveValue('supplier-9')
    expect(screen.getByText(/Suggested from the uploaded document — confirm to assign\./)).toBeInTheDocument()
    expect(screen.queryByLabelText('New supplier name')).not.toBeInTheDocument()
    expect(screen.queryByText(/this purchase order has no supplier/)).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(procurementApi.createSupplierAndAssignToPurchaseOrder).toHaveBeenCalledWith(
      'test-token', 'po-1', { name: 'Pedaços do Brasil, Lda.' },
    ))
  })

  it('still matches when the suggestion has trailing punctuation the stored supplier name lacks', async () => {
    // Reproduces a real report: AI extraction produced "LDA." while the
    // supplier was manually saved as "LDA" (no period).
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier({ id: 'supplier-9', name: 'PEDAÇOS DO BRASIL, LDA' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ supplier_id: null, status: 'needs_review', suggested_supplier_name: 'PEDAÇOS DO BRASIL, LDA.' }),
      line_items: [],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Missing supplier'))

    expect(await screen.findByLabelText('Supplier')).toHaveValue('supplier-9')
    expect(screen.queryByLabelText('New supplier name')).not.toBeInTheDocument()
  })
})

describe('Receiving', () => {
  beforeEach(() => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ line_items: [lineItem({ sku_id: 'sku-1' })] }))
  })

  it('uploads an invoice and shows a parsing indicator', async () => {
    vi.mocked(procurementApi.uploadInvoice).mockResolvedValue({
      id: 'receiving-job-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
      status: 'pending', error_message: '', created_at: '',
    })
    // Never resolves within the test -- keeps the job in-flight so the
    // "parsing" indicator stays visible long enough to assert on.
    vi.mocked(procurementApi.getReceivingParsingJob).mockImplementation(() => new Promise(() => {}))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['sku,qty\nA,8'], 'invoice.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Upload invoice'), file)

    expect(await screen.findByText('Parsing invoice…')).toBeInTheDocument()
    await waitFor(() => expect(procurementApi.uploadInvoice).toHaveBeenCalledWith('test-token', 'po-1', file))
  })

  it('surfaces the invoice parsing job error instead of failing silently', async () => {
    vi.mocked(procurementApi.uploadInvoice).mockResolvedValue({
      id: 'receiving-job-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getReceivingParsingJob).mockResolvedValue({
      id: 'receiving-job-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
      status: 'failed', error_message: 'AI document parsing is not configured', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    await userEvent.upload(screen.getByLabelText('Upload invoice'), file)

    expect(await screen.findByText('AI document parsing is not configured')).toBeInTheDocument()
  })

  it('lists receiving records and expands one to show discrepancies', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail({
      discrepancies: [{
        id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
        line_item_id: 'line-item-1', type: 'quantity_mismatch', expected_qty: 10, received_qty: 8,
        details: 'expected 10, invoice shows 8', status: 'open', created_at: '',
      }],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')

    await userEvent.click(screen.getByText('invoice.csv'))

    expect(await screen.findByText('Quantity mismatch')).toBeInTheDocument()
    expect(screen.getByText('expected 10, invoice shows 8')).toBeInTheDocument()
  })

  it('shows a possible duplicate badge for a flagged receiving record', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord({ is_potential_duplicate: true })])

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Possible duplicate')).toBeInTheDocument()
  })

  it('does not show a duplicate badge for a normal receiving record', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord({ is_potential_duplicate: false })])

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')

    expect(screen.queryByText('Possible duplicate')).not.toBeInTheDocument()
  })

  it('receives stock for a discrepancy line linked to a SKU', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail({
      discrepancies: [{
        id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
        line_item_id: 'line-item-1', type: 'quantity_mismatch', expected_qty: 10, received_qty: 8,
        details: 'expected 10, invoice shows 8', status: 'open', created_at: '',
      }],
    }))
    vi.mocked(inventoryApi.receiveStock).mockResolvedValue({
      batch: null,
      movement: {
        id: 'movement-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'warehouse-1',
        batch_id: null, type: 'receipt', quantity_delta: 8, reason: '', occurred_at: '', created_at: '',
      },
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Receive stock/ }))
    await userEvent.click(await screen.findByRole('button', { name: 'Save receipt' }))

    await waitFor(() => expect(inventoryApi.receiveStock).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      warehouse_id: 'warehouse-1', quantity: 8,
    })))
    expect(await screen.findByText('Stock received.')).toBeInTheDocument()
  })

  // Regression: the discrepancy's received_qty (and the PO line's own
  // Quantity it's diffed against) are in purchased-item/box units -- same
  // as the scanner's "Quantity to receive" bug, submitting the pre-filled
  // default as-is under-received stock by units_per_purchased_item when a
  // line's box isn't its own catalog SKU.
  it('scales the received quantity by the line item\'s units_per_purchased_item before booking stock', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      line_items: [lineItem({ sku_id: 'sku-1', quantity: 5, units_per_purchased_item: 10 })],
    }))
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail({
      discrepancies: [{
        id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
        line_item_id: 'line-item-1', type: 'missing_item', expected_qty: 5, received_qty: 5,
        details: 'expected 5, invoice shows 5', status: 'open', created_at: '',
      }],
    }))
    vi.mocked(inventoryApi.receiveStock).mockResolvedValue({
      batch: null,
      movement: {
        id: 'movement-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'warehouse-1',
        batch_id: null, type: 'receipt', quantity_delta: 50, reason: '', occurred_at: '', created_at: '',
      },
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Receive stock/ }))

    // The field still shows/defaults to 5 -- boxes, matching the PO line's
    // own Quantity -- the multiply happens on submit, not in the field.
    expect(screen.getByLabelText('Qty')).toHaveValue(5)

    await userEvent.click(screen.getByRole('button', { name: 'Save receipt' }))

    await waitFor(() => expect(inventoryApi.receiveStock).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      warehouse_id: 'warehouse-1', quantity: 50,
    })))
  })

  it('prompts for a warehouse when the PO has none yet, assigns it, then receives stock', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ warehouse_id: null })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ warehouse_id: null }), line_items: [lineItem({ sku_id: 'sku-1' })],
    }))
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail({
      discrepancies: [{
        id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
        line_item_id: 'line-item-1', type: 'quantity_mismatch', expected_qty: 10, received_qty: 8,
        details: 'expected 10, invoice shows 8', status: 'open', created_at: '',
      }],
    }))
    vi.mocked(procurementApi.updatePurchaseOrder).mockResolvedValue(po({ warehouse_id: 'warehouse-1' }))
    vi.mocked(inventoryApi.receiveStock).mockResolvedValue({
      batch: null,
      movement: {
        id: 'movement-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'warehouse-1',
        batch_id: null, type: 'receipt', quantity_delta: 8, reason: '', occurred_at: '', created_at: '',
      },
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Receive stock/ }))
    await userEvent.selectOptions(await screen.findByLabelText('Warehouse'), 'warehouse-1')
    await userEvent.click(screen.getByRole('button', { name: 'Save receipt' }))

    await waitFor(() => expect(procurementApi.updatePurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1', expect.objectContaining({
      warehouse_id: 'warehouse-1',
    })))
    await waitFor(() => expect(inventoryApi.receiveStock).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      warehouse_id: 'warehouse-1', quantity: 8,
    })))
    expect(await screen.findByText('Stock received.')).toBeInTheDocument()
  })

  it('completes receiving', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail())
    vi.mocked(procurementApi.completeReceiving).mockResolvedValue(receivingRecord({ status: 'applied' }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Mark receiving complete/ }))

    await waitFor(() => expect(procurementApi.completeReceiving).toHaveBeenCalledWith('test-token', 'receiving-record-1'))
  })

  it('acknowledges a discrepancy and shows it as acknowledged', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    const openDetail = receivingDetail({
      discrepancies: [{
        id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
        line_item_id: 'line-item-1', type: 'quantity_mismatch', expected_qty: 10, received_qty: 8,
        details: 'expected 10, invoice shows 8', status: 'open', created_at: '',
      }],
    })
    const acknowledgedDetail = receivingDetail({
      discrepancies: [{ ...openDetail.discrepancies[0], status: 'acknowledged', acknowledged_at: '2026-08-22T00:00:00Z' }],
    })
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValueOnce(openDetail).mockResolvedValue(acknowledgedDetail)
    vi.mocked(procurementApi.acknowledgeReceivingDiscrepancy).mockResolvedValue(acknowledgedDetail.discrepancies[0])

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Mark as valid/ }))

    await waitFor(() => expect(procurementApi.acknowledgeReceivingDiscrepancy).toHaveBeenCalledWith('test-token', 'discrepancy-1'))
    expect(await screen.findByText('Acknowledged')).toBeInTheDocument()
  })

  it('links an unexpected item to a missing item', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail({
      discrepancies: [
        {
          id: 'unexpected-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
          line_item_id: null, type: 'unexpected_item', expected_qty: 0, received_qty: 5,
          details: 'invoice line "Widget B" was not on the purchase order', status: 'open', created_at: '',
        },
        {
          id: 'missing-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
          line_item_id: 'line-item-1', type: 'missing_item', expected_qty: 5, received_qty: 0,
          details: '"Widget A" was on the purchase order but not on the invoice', status: 'open', created_at: '',
        },
      ],
    }))
    vi.mocked(procurementApi.linkReceivingDiscrepancies).mockResolvedValue(undefined)

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click((await screen.findAllByRole('button', { name: /Link to…/ }))[0])
    await userEvent.selectOptions(await screen.findByLabelText('Link to discrepancy'), 'missing-1')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(procurementApi.linkReceivingDiscrepancies).toHaveBeenCalledWith('test-token', 'unexpected-1', 'missing-1'))
  })

  it('flags a line item as damaged', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail())
    vi.mocked(procurementApi.flagDamagedLineItem).mockResolvedValue({
      id: 'discrepancy-2', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
      line_item_id: 'line-item-1', type: 'damaged_item', expected_qty: 0, received_qty: 2,
      details: 'crushed box', status: 'open', created_at: '',
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Flag damaged item/ }))
    await userEvent.clear(await screen.findByRole('spinbutton'))
    await userEvent.type(screen.getByRole('spinbutton'), '2')
    await userEvent.type(screen.getByPlaceholderText('Note (optional)'), 'crushed box')
    await userEvent.click(screen.getByRole('button', { name: 'Flag as damaged' }))

    await waitFor(() => expect(procurementApi.flagDamagedLineItem).toHaveBeenCalledWith('test-token', 'receiving-record-1', 'line-item-1', 2, 'crushed box'))
  })

  it('generates a discrepancy report grouped by type', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord()])
    vi.mocked(procurementApi.getReceivingRecord).mockResolvedValue(receivingDetail({
      discrepancies: [
        {
          id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
          line_item_id: 'line-item-1', type: 'quantity_mismatch', expected_qty: 10, received_qty: 8,
          details: 'expected 10, invoice shows 8', status: 'open', created_at: '',
        },
        {
          id: 'discrepancy-2', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
          line_item_id: 'line-item-1', type: 'damaged_item', expected_qty: 0, received_qty: 2,
          details: 'crushed box', status: 'open', created_at: '',
        },
      ],
    }))

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Generate report/ }))

    expect(await screen.findByText('Receiving discrepancy report')).toBeInTheDocument()
    expect(screen.getByText((_, el) => el?.tagName === 'H3' && el.textContent === 'Damaged (1)')).toBeInTheDocument()
    expect(screen.getByText((_, el) => el?.tagName === 'H3' && el.textContent === 'Quantity mismatch (1)')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Export CSV/ })).toBeInTheDocument()
  })
})

describe('Booking discrepancies report', () => {
  it('opens from the toolbar and shows missing items by default', async () => {
    vi.mocked(procurementApi.listBookingDiscrepancies).mockResolvedValue({
      missing_items: [{
        booking_id: 'booking-1', purchase_order_id: 'po-1', line_item_id: 'line-1',
        sku_code: 'WGT-1', product_name: 'Widget', raw_description: 'Widget 12pk',
        expected_qty: 10, scanned_qty: 4, missing_qty: 6,
      }],
      damaged_lost: [],
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: /Discrepancies/ }))

    expect(await screen.findByText('Booking discrepancies')).toBeInTheDocument()
    expect(screen.getByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('WGT-1')).toBeInTheDocument()
  })

  it('Damaged / Lost tab shows write-off scans', async () => {
    vi.mocked(procurementApi.listBookingDiscrepancies).mockResolvedValue({
      missing_items: [],
      damaged_lost: [{
        booking_id: 'booking-1', purchase_order_id: 'po-1', scan_id: 'scan-1', sku_id: 'sku-1',
        sku_code: 'WGT-1', product_name: 'Widget', action_type: 'damaged', quantity: 2,
        reason: 'box crushed', scanned_at: '2026-08-15T00:00:00Z', scanned_by: 'user-1',
      }],
    })

    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: /Discrepancies/ }))
    await userEvent.click(await screen.findByRole('button', { name: /Damaged \/ Lost/ }))

    expect(await screen.findByText('box crushed')).toBeInTheDocument()
    expect(screen.getByText('damaged')).toBeInTheDocument()
    expect(screen.getByText('user-1')).toBeInTheDocument()
  })

  it('Back returns to the purchase order list', async () => {
    renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: /Discrepancies/ }))
    await screen.findByText('Booking discrepancies')

    await userEvent.click(screen.getByRole('button', { name: /Procurement/ }))

    expect(await screen.findByRole('button', { name: /Discrepancies/ })).toBeInTheDocument()
  })
})

describe('Procurement URLs (refresh + Back)', () => {
  it('opening a PO pushes /procurement/:id and Back returns to the list', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ id: 'po-1' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-1' }) }))
    const { router } = renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByText('Acme Foods'))
    expect(router.state.location.pathname).toBe('/procurement/po-1')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/procurement')
    expect(await screen.findByRole('button', { name: /Add purchase order/ })).toBeInTheDocument()
  })

  it('/procurement/:id loads that PO on refresh', async () => {
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-1' }) }))
    renderAt(<Procurement />, '/procurement', '/procurement/po-1')
    await waitFor(() => expect(procurementApi.getPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1'))
    expect(await screen.findByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('creating from /procurement/new replaces the URL with the new PO id', async () => {
    vi.mocked(procurementApi.createPurchaseOrder).mockResolvedValue(po({ id: 'po-new' }))
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-new' }) }))
    const { router } = renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: /Add purchase order/ }))
    expect(router.state.location.pathname).toBe('/procurement/new')

    await userEvent.selectOptions(await screen.findByLabelText('Supplier'), 'supplier-1')
    await userEvent.click(screen.getByRole('button', { name: 'Create purchase order' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/procurement/po-new'))
    expect(router.state.historyAction).toBe('REPLACE')
  })

  it('an upload-created PO carries its parse job in the URL, so a refresh keeps polling it', async () => {
    vi.mocked(procurementApi.createPurchaseOrderFromDocument).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-new', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-new' }) }))
    vi.mocked(procurementApi.getDocumentParsingJob).mockImplementation(() => new Promise(() => {}))

    const { router } = renderAt(<Procurement />, '/procurement')
    await userEvent.click(await screen.findByRole('button', { name: 'Upload document' }))
    await userEvent.upload(await screen.findByLabelText('Document'), new File(['a'], 'order.csv', { type: 'text/csv' }))
    await userEvent.click(screen.getByRole('button', { name: 'Upload & create' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/procurement/po-new'))
    expect(router.state.location.search).toBe('?parsing=job-1')
  })

  it('the status filter lives in ?status=', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ id: 'po-1', status: 'draft' }), po({ id: 'po-2', status: 'sent' })])
    renderAt(<Procurement />, '/procurement', '/procurement?status=sent')
    expect(await screen.findByText('1 purchase order')).toBeInTheDocument()
  })

  it('/procurement/discrepancies opens the discrepancies page, tab in ?tab=', async () => {
    const { router } = renderAt(<Procurement />, '/procurement', '/procurement/discrepancies?tab=damaged-lost')
    expect(await screen.findByText('Booking discrepancies')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Missing/ }))
    expect(router.state.location.search).toBe('')
  })
})
