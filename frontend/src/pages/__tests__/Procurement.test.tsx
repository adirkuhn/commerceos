// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Procurement from '../Procurement'
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
  createPurchaseOrderFromDocument: vi.fn(),
  linkLineItemToSku: vi.fn(),
  createProductAndLinkLineItem: vi.fn(),
  createSupplierAndAssignToPurchaseOrder: vi.fn(),
  uploadInvoice: vi.fn(),
  getReceivingParsingJob: vi.fn(),
  listReceivingRecords: vi.fn(),
  getReceivingRecord: vi.fn(),
  completeReceiving: vi.fn(),
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

vi.mock('../../scanner/api', () => ({
  generatePairingCode: vi.fn(),
  getPairingCodeStatus: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  createProduct: vi.fn(),
  updateProduct: vi.fn(),
  deleteProduct: vi.fn(),
}))

vi.mock('../../integrations/api', () => ({
  getAIConnection: vi.fn(),
}))

import * as procurementApi from '../../procurement/api'
import * as suppliersApi from '../../suppliers/api'
import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../../pms/api'
import * as scannerApi from '../../scanner/api'
import * as settingsApi from '../../settings/api'
import * as integrationsApi from '../../integrations/api'

function po(overrides: Partial<PurchaseOrder> = {}): PurchaseOrder {
  return {
    id: 'po-1', tenant_id: 'tenant-1', supplier_id: 'supplier-1', warehouse_id: 'warehouse-1',
    status: 'draft', currency_code: 'USD', expected_date: null, notes: '',
    is_potential_duplicate: false,
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
    raw_description: 'Widget A', quantity: 10, unit_cost_minor_units: 500,
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
  vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
  // Empty default_currency means the "no tenant setting yet" fallback (USD)
  // applies -- individual tests override this to verify the real default.
  vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '', updated_at: '',
  })
  // Configured by default so existing PDF-upload tests aren't interrupted
  // by the "no AI configured" confirm prompt -- tests for that prompt
  // specifically override this to null.
  vi.mocked(integrationsApi.getAIConnection).mockResolvedValue({
    id: 'ai-conn-1', provider: 'openai', status: 'connected', created_at: '',
  })
})

describe('Purchase orders list', () => {
  it('shows an empty state when there are no purchase orders', async () => {
    render(<Procurement />)
    expect(await screen.findByText('No purchase orders yet')).toBeInTheDocument()
  })

  it('lists purchase orders resolving supplier and warehouse names', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])

    render(<Procurement />)

    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
    expect(screen.getByText('Main Warehouse')).toBeInTheDocument()
    expect(screen.getByText('draft', { selector: 'span' })).toBeInTheDocument()
  })

  it('filters by status tab', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([
      po({ id: 'po-1', status: 'draft' }),
      po({ id: 'po-2', status: 'sent', supplier_id: 'supplier-1' }),
    ])

    render(<Procurement />)
    await screen.findAllByText('Acme Foods')
    expect(screen.getByText((_, node) => node?.tagName === 'SPAN' && node.textContent === '2 purchase orders')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'sent' }))

    expect(screen.getByText((_, node) => node?.tagName === 'SPAN' && node.textContent === '1 purchase order')).toBeInTheDocument()
  })

  it('deletes a purchase order after confirming', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<Procurement />)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(procurementApi.deletePurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1'))
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.stubGlobal('confirm', vi.fn(() => false))

    render(<Procurement />)
    await userEvent.click(await screen.findByTitle('Delete'))

    expect(procurementApi.deletePurchaseOrder).not.toHaveBeenCalled()
  })

  it('shows a duplicate-order badge and opens the other PO when clicked', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([
      po({ id: 'po-2', is_potential_duplicate: true, duplicate_of_purchase_order_id: 'po-1' }),
    ])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ purchase_order: po({ id: 'po-1' }) }))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Possible duplicate order'))

    await waitFor(() => expect(procurementApi.getPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1'))
  })

  it('does not show a duplicate-order badge for a normal PO', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])

    render(<Procurement />)
    await screen.findByText('Acme Foods')

    expect(screen.queryByText('Possible duplicate order')).not.toBeInTheDocument()
  })
})

describe('Purchase order create', () => {
  it('opening Add Purchase Order shows a blank create form with no line items section', async () => {
    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Purchase Order' }))

    expect(screen.getByText('New Purchase Order', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create Purchase Order' })).toBeInTheDocument()
    expect(screen.queryByText('Line Items')).not.toBeInTheDocument()
  })

  it('creates a new purchase order and returns to the list', async () => {
    vi.mocked(procurementApi.createPurchaseOrder).mockResolvedValue(po())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail())

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Purchase Order' }))
    await userEvent.selectOptions(screen.getByLabelText('Supplier'), 'supplier-1')
    await userEvent.selectOptions(screen.getByLabelText('Warehouse (optional)'), 'warehouse-1')
    await userEvent.click(screen.getByRole('button', { name: 'Create Purchase Order' }))

    await waitFor(() => expect(procurementApi.createPurchaseOrder).toHaveBeenCalledWith('test-token', expect.objectContaining({
      supplier_id: 'supplier-1', warehouse_id: 'warehouse-1', currency_code: 'USD',
      expected_date: new Date().toISOString().slice(0, 10),
    })))
  })

  it('defaults currency to a select with USD chosen and expected date to today', async () => {
    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Purchase Order' }))

    expect(screen.getByLabelText('Currency').tagName).toBe('SELECT')
    expect(screen.getByLabelText('Currency')).toHaveValue('USD')
    expect(screen.getByLabelText('Expected date')).toHaveValue(new Date().toISOString().slice(0, 10))
  })

  it('defaults currency to the tenant system-settings default once it loads', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '', updated_at: '',
    })

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Purchase Order' }))

    await waitFor(() => expect(screen.getByLabelText('Currency')).toHaveValue('EUR'))
  })

  it('does not override a currency the user already picked once settings load', async () => {
    let resolveSettings: (v: import('../../settings/types').TenantSettings) => void
    vi.mocked(settingsApi.getTenantSettings).mockReturnValue(new Promise(resolve => { resolveSettings = resolve }))

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Purchase Order' }))
    await userEvent.selectOptions(screen.getByLabelText('Currency'), 'GBP')

    resolveSettings!({ tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '', updated_at: '' })

    // Give the settings promise's .then() a tick to run before asserting —
    // still wrapped in waitFor so the resulting state update is act()-safe.
    await waitFor(() => expect(screen.getByLabelText('Currency')).toHaveValue('GBP'))
  })
})

describe('Upload document to create a PO', () => {
  it('opens a modal with just a file field -- no supplier, warehouse, currency, or date to fill in', async () => {
    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Upload Document' }))

    expect(screen.getByText('Upload Document', { selector: 'h2' })).toBeInTheDocument()
    expect(screen.getByLabelText('Document')).toBeInTheDocument()
    expect(screen.queryByLabelText(/Supplier/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Warehouse/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Currency/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/date/i)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Upload & Create' })).toBeDisabled()
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

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Upload Document' }))
    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Document'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload & Create' }))

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

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Upload Document' }))
    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Document'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload & Create' }))

    expect(await screen.findByText('Parsing document…')).toBeInTheDocument()
    expect(procurementApi.getDocumentParsingJob).toHaveBeenCalledWith('test-token', 'job-1')
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

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Upload Document' }))
    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Document'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload & Create' }))

    // No second upload needed to see what the first one extracted -- the
    // PO page itself picks it up once the background parse completes.
    expect(await screen.findByText('Widget A')).toBeInTheDocument()
  })

  it('shows an error if the upload fails', async () => {
    vi.mocked(procurementApi.createPurchaseOrderFromDocument).mockRejectedValue(new Error('document is missing required columns'))

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: 'Upload Document' }))
    await userEvent.upload(screen.getByLabelText('Document'), new File(['bad'], 'bad.csv', { type: 'text/csv' }))
    await userEvent.click(screen.getByRole('button', { name: 'Upload & Create' }))

    expect(await screen.findByText('document is missing required columns')).toBeInTheDocument()
  })
})

describe('Purchase order edit with line items', () => {
  beforeEach(() => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({ line_items: [lineItem()] }))
  })

  it('prefills header fields and shows existing line items', async () => {
    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('10')).toBeInTheDocument()
    expect(screen.getByLabelText('Supplier')).toHaveValue('supplier-1')
  })

  it('adds a new line item', async () => {
    vi.mocked(procurementApi.addLineItem).mockResolvedValue(lineItem({ id: 'line-item-2', raw_description: 'Widget B', quantity: 5 }))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    await userEvent.click(screen.getByRole('button', { name: /Add Line Item/ }))
    await userEvent.type(screen.getByLabelText('Line item description'), 'Widget B')
    await userEvent.clear(screen.getByLabelText('Quantity'))
    await userEvent.type(screen.getByLabelText('Quantity'), '5')
    await userEvent.clear(screen.getByLabelText('Unit cost'))
    await userEvent.type(screen.getByLabelText('Unit cost'), '12.50')
    await userEvent.click(screen.getByTitle('Save'))

    await waitFor(() => expect(procurementApi.addLineItem).toHaveBeenCalledWith('test-token', 'po-1', {
      sku_id: null, raw_description: 'Widget B', quantity: 5, unit_cost_minor_units: 1250,
    }))
  })

  it('removes a line item after confirming', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    const row = (await screen.findByText('Widget A')).closest('tr')!
    await userEvent.click(within(row).getByTitle('Remove'))

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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    await userEvent.click(screen.getByRole('button', { name: /Add Line Item/ }))
    await userEvent.click(screen.getByRole('button', { name: /Link product/ }))
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Widget')
    await userEvent.click(await screen.findByText('Widget Product'))
    await userEvent.click(await screen.findByText('WID-001'))

    expect(screen.queryByPlaceholderText('Search products by name…')).not.toBeInTheDocument()
    await userEvent.click(screen.getByTitle('Save'))

    await waitFor(() => expect(procurementApi.addLineItem).toHaveBeenCalledWith('test-token', 'po-1', expect.objectContaining({
      sku_id: 'sku-1',
    })))
  })

  it('Discard returns to the list without saving', async () => {
    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(procurementApi.updatePurchaseOrder).not.toHaveBeenCalled()
    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
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

    render(<Procurement />)
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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    expect(await screen.findByText(/add line items manually/)).toBeInTheDocument()
    expect(screen.queryByText('AI document parsing is not configured')).not.toBeInTheDocument()
  })

  it('warns before uploading a PDF when no AI provider is configured, and still uploads if confirmed', async () => {
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue(null)
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.mocked(procurementApi.attachDocumentToPurchaseOrder).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'completed', error_message: '', created_at: '',
    })

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    expect(window.confirm).toHaveBeenCalled()
    await waitFor(() => expect(procurementApi.attachDocumentToPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1', file))
  })

  it('does not upload a PDF when the no-AI warning is dismissed', async () => {
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue(null)
    vi.stubGlobal('confirm', vi.fn(() => false))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['%PDF-1.4'], 'invoice.pdf', { type: 'application/pdf' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    expect(window.confirm).toHaveBeenCalled()
    expect(procurementApi.attachDocumentToPurchaseOrder).not.toHaveBeenCalled()
  })

  it('does not warn before uploading a CSV, since it never needs AI', async () => {
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue(null)
    const confirmSpy = vi.fn(() => true)
    vi.stubGlobal('confirm', confirmSpy)
    vi.mocked(procurementApi.attachDocumentToPurchaseOrder).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'pending', error_message: '', created_at: '',
    })
    vi.mocked(procurementApi.getDocumentParsingJob).mockResolvedValue({
      id: 'job-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', document_id: 'document-1',
      status: 'completed', error_message: '', created_at: '',
    })

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Widget A')

    const file = new File(['sku,qty\nA,1'], 'order.csv', { type: 'text/csv' })
    const fileInput = document.querySelector('input[type="file"][accept=".pdf,.csv,.xlsx"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    await waitFor(() => expect(procurementApi.attachDocumentToPurchaseOrder).toHaveBeenCalledWith('test-token', 'po-1', file))
    expect(confirmSpy).not.toHaveBeenCalled()
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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))

    const link = await screen.findByText('invoice.pdf')
    expect(link.closest('a')).toHaveAttribute('href', 'https://uploads.example.com/tenant-1/abc.pdf')
  })

  it('does not render a Documents section when there are none', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail())

    render(<Procurement />)
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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Possible duplicate')).toBeInTheDocument()
  })

  it('deletes a document after confirming', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))
    vi.mocked(procurementApi.deletePurchaseOrderDocument).mockResolvedValue(null)

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Delete'))

    await waitFor(() => expect(procurementApi.deletePurchaseOrderDocument).toHaveBeenCalledWith('test-token', 'document-1'))
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false))
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      documents: [{
        id: 'document-1', purchase_order_id: 'po-1', file_url: 'https://uploads.example.com/tenant-1/abc.pdf',
        file_name: 'invoice.pdf', content_type: 'application/pdf', is_potential_duplicate: false, created_at: '',
      }],
    }))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.pdf')
    await userEvent.click(screen.getByTitle('Delete'))

    expect(procurementApi.deletePurchaseOrderDocument).not.toHaveBeenCalled()
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
    render(<Procurement />)
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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.click(await screen.findByText('Needs review'))
    await userEvent.click(screen.getByRole('button', { name: /Link to existing product/ }))
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Widget')
    await userEvent.click(await screen.findByText('Widget Product'))
    await userEvent.click(await screen.findByText('WID-001'))

    await waitFor(() => expect(procurementApi.linkLineItemToSku).toHaveBeenCalledWith('test-token', 'line-item-unmatched', { sku_id: 'sku-1' }))
  })

  it('creates a new product and links an unmatched item', async () => {
    vi.mocked(procurementApi.createProductAndLinkLineItem).mockResolvedValue(lineItem({ id: 'line-item-unmatched', sku_id: 'new-sku-1' }))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.click(await screen.findByText('Needs review'))
    await userEvent.click(screen.getByRole('button', { name: /Create new product/ }))

    const nameInput = screen.getByDisplayValue('Mystery Item')
    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, 'Mystery Widget')
    await userEvent.click(screen.getByRole('button', { name: 'Create & Link' }))

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
    render(<Procurement />)

    expect(await screen.findByText('Missing supplier')).toBeInTheDocument()

    await userEvent.click(screen.getByText('Missing supplier'))

    expect(await screen.findByText(/this purchase order has no supplier/)).toBeInTheDocument()
  })

  it('creates a supplier and assigns it in one click', async () => {
    vi.mocked(procurementApi.createSupplierAndAssignToPurchaseOrder).mockResolvedValue(
      po({ supplier_id: 'new-supplier-1', status: 'draft' }),
    )

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Missing supplier'))
    await userEvent.type(screen.getByLabelText('New supplier name'), 'Brand New Supplier')
    await userEvent.click(screen.getByRole('button', { name: 'Create & Assign' }))

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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Missing supplier'))

    const nameInput = await screen.findByLabelText('New supplier name')
    expect(nameInput).toHaveValue('Pedaços do Brasil, Lda.')

    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, 'Corrected Name')
    await userEvent.click(screen.getByRole('button', { name: 'Create & Assign' }))

    await waitFor(() => expect(procurementApi.createSupplierAndAssignToPurchaseOrder).toHaveBeenCalledWith(
      'test-token', 'po-1', { name: 'Corrected Name' },
    ))
  })

  it('offers to assign, not create, when the suggested name matches an existing supplier', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier({ name: 'Pedaços do Brasil, Lda.' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue(detail({
      purchase_order: po({ supplier_id: null, status: 'needs_review', suggested_supplier_name: 'PEDAÇOS DO BRASIL, LDA.' }),
      line_items: [],
    }))
    vi.mocked(procurementApi.createSupplierAndAssignToPurchaseOrder).mockResolvedValue(
      po({ supplier_id: 'supplier-1', status: 'draft' }),
    )

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Missing supplier'))

    expect(await screen.findByText(/Matches existing supplier "Pedaços do Brasil, Lda\."/)).toBeInTheDocument()
    const assignButton = screen.getByRole('button', { name: 'Assign existing supplier' })
    expect(assignButton).toBeInTheDocument()

    await userEvent.click(assignButton)

    await waitFor(() => expect(procurementApi.createSupplierAndAssignToPurchaseOrder).toHaveBeenCalledWith(
      'test-token', 'po-1', { name: 'PEDAÇOS DO BRASIL, LDA.' },
    ))
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

    render(<Procurement />)
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

    render(<Procurement />)
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
        details: 'expected 10, invoice shows 8', created_at: '',
      }],
    }))

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')

    await userEvent.click(screen.getByText('invoice.csv'))

    expect(await screen.findByText('Quantity mismatch')).toBeInTheDocument()
    expect(screen.getByText('expected 10, invoice shows 8')).toBeInTheDocument()
  })

  it('shows a possible duplicate badge for a flagged receiving record', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord({ is_potential_duplicate: true })])

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Possible duplicate')).toBeInTheDocument()
  })

  it('does not show a duplicate badge for a normal receiving record', async () => {
    vi.mocked(procurementApi.listReceivingRecords).mockResolvedValue([receivingRecord({ is_potential_duplicate: false })])

    render(<Procurement />)
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
        details: 'expected 10, invoice shows 8', created_at: '',
      }],
    }))
    vi.mocked(inventoryApi.receiveStock).mockResolvedValue({
      batch: null,
      movement: {
        id: 'movement-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'warehouse-1',
        batch_id: null, type: 'receipt', quantity_delta: 8, reason: '', occurred_at: '', created_at: '',
      },
    })

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Receive stock/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Save receipt' }))

    await waitFor(() => expect(inventoryApi.receiveStock).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      warehouse_id: 'warehouse-1', quantity: 8,
    })))
    expect(await screen.findByText('Stock received.')).toBeInTheDocument()
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
        details: 'expected 10, invoice shows 8', created_at: '',
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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Receive stock/ }))
    await userEvent.selectOptions(screen.getByLabelText('Warehouse'), 'warehouse-1')
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

    render(<Procurement />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('invoice.csv')
    await userEvent.click(screen.getByText('invoice.csv'))
    await userEvent.click(await screen.findByRole('button', { name: /Mark Receiving Complete/ }))

    await waitFor(() => expect(procurementApi.completeReceiving).toHaveBeenCalledWith('test-token', 'receiving-record-1'))
  })
})

describe('Pair Scanner', () => {
  it('generates a pairing code and shows it', async () => {
    vi.mocked(scannerApi.generatePairingCode).mockResolvedValue({
      id: 'pairing-1', tenant_id: 'tenant-1', code: '654321', status: 'pending',
      expires_at: '', created_at: '',
    })

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: /Pair Scanner/ }))

    expect(await screen.findByText('654321')).toBeInTheDocument()
    expect(screen.getByText('Waiting for scanner…')).toBeInTheDocument()
  })

  it('closes the modal', async () => {
    vi.mocked(scannerApi.generatePairingCode).mockResolvedValue({
      id: 'pairing-1', tenant_id: 'tenant-1', code: '654321', status: 'pending',
      expires_at: '', created_at: '',
    })

    render(<Procurement />)
    await userEvent.click(await screen.findByRole('button', { name: /Pair Scanner/ }))
    await screen.findByText('654321')
    await userEvent.click(screen.getByTitle('Close'))

    expect(screen.queryByText('654321')).not.toBeInTheDocument()
  })
})
