// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import SalesOrders from '../SalesOrders'
import type { SalesOrder, SalesOrderDetail } from '../../oms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  getSalesOrder: vi.fn(),
  listSalesOrderActivities: vi.fn(),
  createSalesOrder: vi.fn(),
  updateSalesOrder: vi.fn(),
  deleteSalesOrder: vi.fn(),
  bulkUpdateSalesOrders: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
  updateTenantSettings: vi.fn(),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
}))

vi.mock('../../shipping/api', () => ({
  exportMyParcelOrder: vi.fn(),
  createShipment: vi.fn(),
  getShippingSettings: vi.fn(),
  getShipmentLabel: vi.fn(),
}))

vi.mock('../../tax/api', () => ({
  listTaxClasses: vi.fn(),
  createTaxClass: vi.fn(),
  updateTaxClass: vi.fn(),
  deleteTaxClass: vi.fn(),
  listTaxRates: vi.fn(),
  upsertTaxRate: vi.fn(),
  deleteTaxRate: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../../pms/api'
import * as tenantSettingsApi from '../../settings/api'
import * as taxApi from '../../tax/api'
import * as integrationsApi from '../../integrations/api'
import * as shippingApi from '../../shipping/api'

function warehouse(overrides: Record<string, unknown> = {}) {
  return {
    id: 'wh-1', tenant_id: 'tenant-1', name: 'Main Warehouse',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'pos-desktop', external_order_id: 'ext-1',
    number: '1001', order_key: 'wc_order_abc', status: 'completed', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    ...overrides,
  }
}

function detail(overrides: Partial<SalesOrderDetail> = {}): SalesOrderDetail {
  return {
    sales_order: salesOrder(),
    line_items: [{
      id: 'li-1', sku_id: 'sku-1', name: 'Blue Widget', sku_code: 'WID-BLU', barcode: '5601049213991',
      quantity: 2, price_minor_units: 2000, subtotal_minor_units: 4000, subtotal_tax_minor_units: 0,
      total_minor_units: 4000, total_tax_minor_units: 0,
    }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: 'USD', timezone: '', date_format: '', language: '',
    prices_include_tax: true, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listTaxRates).mockResolvedValue([])
  vi.mocked(omsApi.bulkUpdateSalesOrders).mockResolvedValue({ updated: [], failed: [] })
  vi.mocked(omsApi.listSalesOrderActivities).mockResolvedValue({ activities: [] })
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
  vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
})

describe('Sales orders list', () => {
  it('shows an empty state when there are no sales orders', async () => {
    render(<SalesOrders />)

    expect(await screen.findByText('No sales orders yet')).toBeInTheDocument()
  })

  it('lists sales orders with customer, status, and total', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    render(<SalesOrders />)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('completed')).toBeInTheDocument()
    expect(screen.getByText('$50.00')).toBeInTheDocument()
  })

  it('shows a fulfilled/unfulfilled badge per order', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001', fulfilled: true }), salesOrder({ id: 'so-2', number: '1002', fulfilled: false })],
      total: 2,
    })

    render(<SalesOrders />)
    await screen.findByText('#1001')

    expect(screen.getByText('Fulfilled')).toBeInTheDocument()
    expect(screen.getByText('Unfulfilled')).toBeInTheDocument()
  })

  it('paginates using the server page param', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 25 })

    render(<SalesOrders />)
    await screen.findByText('#1001')

    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', { page: 2, perPage: 20, sortBy: 'occurred_at', sortDir: 'desc' })
  })

  it('defaults to sorting by date, most recent first', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    render(<SalesOrders />)
    await screen.findByText('#1001')

    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', { page: 1, perPage: 20, sortBy: 'occurred_at', sortDir: 'desc' })
  })

  it('clicking a column header sorts by that field', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    render(<SalesOrders />)
    await screen.findByText('#1001')

    await userEvent.click(screen.getByRole('columnheader', { name: /Total/ }))

    expect(omsApi.listSalesOrders).toHaveBeenLastCalledWith('test-token', { page: 1, perPage: 20, sortBy: 'total', sortDir: 'desc' })
  })

  it('clicking the active column header again flips the direction', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    render(<SalesOrders />)
    await screen.findByText('#1001')

    await userEvent.click(screen.getByRole('columnheader', { name: /Order/ }))

    expect(omsApi.listSalesOrders).toHaveBeenLastCalledWith('test-token', { page: 1, perPage: 20, sortBy: 'occurred_at', sortDir: 'asc' })
  })

  it('changing sort resets back to page 1', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 25 })

    render(<SalesOrders />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))
    await waitFor(() => expect(omsApi.listSalesOrders).toHaveBeenLastCalledWith('test-token', { page: 2, perPage: 20, sortBy: 'occurred_at', sortDir: 'desc' }))

    await userEvent.click(screen.getByRole('columnheader', { name: /Customer/ }))

    expect(omsApi.listSalesOrders).toHaveBeenLastCalledWith('test-token', { page: 1, perPage: 20, sortBy: 'customer', sortDir: 'desc' })
  })
})

describe('Sales orders bulk actions', () => {
  it('selecting a row reveals the bulk action bar, applying calls the API for the selected ids after confirmation', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001' }), salesOrder({ id: 'so-2', number: '1002' })], total: 2,
    })
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<SalesOrders />)
    await screen.findByText('#1001')

    expect(screen.queryByText('2 selected')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select order #1001' }))
    expect(screen.getByText('1 selected')).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Bulk change status' }), 'processing')
    await userEvent.click(screen.getByRole('button', { name: 'Apply' }))

    await waitFor(() => expect(omsApi.bulkUpdateSalesOrders).toHaveBeenCalledWith('test-token', {
      orderIds: ['so-1'], status: 'processing', fulfilled: undefined,
    }))
    // selection clears and the bar disappears after a successful apply
    await waitFor(() => expect(screen.queryByText('1 selected')).not.toBeInTheDocument())
  })

  it('does not apply when the confirm dialog is dismissed', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder({ id: 'so-1', number: '1001' })], total: 1 })
    vi.stubGlobal('confirm', vi.fn(() => false))

    render(<SalesOrders />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select order #1001' }))
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Bulk change fulfilled' }), 'true')
    await userEvent.click(screen.getByRole('button', { name: 'Apply' }))

    expect(omsApi.bulkUpdateSalesOrders).not.toHaveBeenCalled()
  })

  it('selecting a row does not open the order detail page', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder({ id: 'so-1', number: '1001' })], total: 1 })

    render(<SalesOrders />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select order #1001' }))

    expect(omsApi.getSalesOrder).not.toHaveBeenCalled()
  })

  it('the select-all checkbox selects every order on the current page', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001' }), salesOrder({ id: 'so-2', number: '1002' })], total: 2,
    })

    render(<SalesOrders />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select all orders on this page' }))

    expect(screen.getByText('2 selected')).toBeInTheDocument()
  })

  it('reports partial failures without discarding the successful updates', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder({ id: 'so-1', number: '1001' })], total: 1 })
    vi.mocked(omsApi.bulkUpdateSalesOrders).mockResolvedValue({ updated: [], failed: [{ orderId: 'so-1', error: 'sales order not found' }] })
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<SalesOrders />)
    await screen.findByText('#1001')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select order #1001' }))
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Bulk change status' }), 'cancelled')
    await userEvent.click(screen.getByRole('button', { name: 'Apply' }))

    expect(await screen.findByText(/sales order not found/)).toBeInTheDocument()
  })
})

describe('Sales order detail page', () => {
  it('opens the detail page and shows line items when clicking a row', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Blue Widget')).toBeInTheDocument()
    expect(screen.getByText('WID-BLU')).toBeInTheDocument()
    expect(screen.getByText('5601049213991')).toBeInTheDocument()
    expect(omsApi.getSalesOrder).toHaveBeenCalledWith('test-token', 'so-1')
  })

  it('shows the order activity log when it has entries', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    vi.mocked(omsApi.listSalesOrderActivities).mockResolvedValue({
      activities: [{
        id: 'a-1', sales_order_id: 'so-1', user_id: 'user-1', activity_type: 'myparcel_shipment_created',
        summary: 'Shipment created via MyParcel', metadata: {}, created_at: '2026-08-26T00:00:00Z',
      }],
    })

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Shipment created via MyParcel')).toBeInTheDocument()
  })

  it('does not show Create label when no MyParcel connection exists', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    expect(screen.queryByRole('button', { name: 'Create label' })).not.toBeInTheDocument()
  })

  it('exports the order to MyParcel, creates the shipment, and shows Print label afterward', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([
      { id: 'conn-mp-1', provider: 'myparcel', store_url: '', status: 'connected', created_at: '2026-08-01T00:00:00Z' },
    ])
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({
        shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
      })],
      total: 1,
    })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      sales_order: salesOrder({
        shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
      }),
    }))
    vi.mocked(shippingApi.exportMyParcelOrder).mockResolvedValue({
      id: 'export-1', connection_id: 'conn-mp-1', sales_order_id: 'so-1', myparcel_uuid: 'order-uuid-1', status: 'open',
    })
    vi.mocked(shippingApi.createShipment).mockResolvedValue({
      id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 1, barcode: '3SMYPA1',
      status: 1, status_main: 'registered', label_format: 'pdf',
      recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
      created_at: '2026-08-23T00:00:00Z',
    })

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByRole('button', { name: 'Create label' }))
    expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Create shipping label' }))

    await waitFor(() => expect(shippingApi.exportMyParcelOrder).toHaveBeenCalledWith('test-token', 'conn-mp-1', 'so-1', expect.objectContaining({ carrier: 1 })))
    expect(shippingApi.createShipment).toHaveBeenCalledWith('test-token', expect.objectContaining({ connection_id: 'conn-mp-1', sales_order_id: 'so-1' }))
    expect(await screen.findByRole('button', { name: 'Print label' })).toBeInTheDocument()
  })

  it('auto-opens the print view when auto_print_enabled is on, and offers Retry print', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([
      { id: 'conn-mp-1', provider: 'myparcel', store_url: '', status: 'connected', created_at: '2026-08-01T00:00:00Z' },
    ])
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({
      settings: {
        connection_id: 'conn-mp-1', default_carrier: 1, default_package_type: 1, default_label_format: 'pdf',
        default_signature_required: false, default_only_recipient: false, insure_by_default: false,
        myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: true,
      },
      carrier_overrides: [],
    })
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({
        shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
      })],
      total: 1,
    })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      sales_order: salesOrder({
        shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
      }),
    }))
    vi.mocked(shippingApi.exportMyParcelOrder).mockResolvedValue({
      id: 'export-1', connection_id: 'conn-mp-1', sales_order_id: 'so-1', myparcel_uuid: 'order-uuid-1', status: 'open',
    })
    vi.mocked(shippingApi.createShipment).mockResolvedValue({
      id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 1, barcode: '3SMYPA1',
      status: 1, status_main: 'registered', label_format: 'pdf',
      recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
      created_at: '2026-08-23T00:00:00Z',
    })
    // ShippingLabelPrintView fetches the label itself -- give it something
    // that resolves so it doesn't hang on "Loading label…".
    vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(new Blob(['%PDF-fake'], { type: 'application/pdf' }))
    vi.stubGlobal('print', vi.fn())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByRole('button', { name: 'Create label' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Create shipping label' }))

    // The print view opens on its own -- no click needed to get here.
    expect(await screen.findByTitle('Close')).toBeInTheDocument()
    await userEvent.click(screen.getByTitle('Close'))

    expect(await screen.findByRole('button', { name: 'Retry print' })).toBeInTheDocument()
  })

  it('shows no activity section when the log is empty', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    expect(screen.queryByText('Activity')).not.toBeInTheDocument()
  })

  it('Picking list opens the printable view and triggers the browser print dialog', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    const printSpy = vi.fn()
    vi.stubGlobal('print', printSpy)

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByTitle('Print picking list'))

    expect(await screen.findByText('Picking List')).toBeInTheDocument()
    expect(printSpy).toHaveBeenCalled()

    await userEvent.click(screen.getByTitle('Close'))
    expect(screen.getByText('Blue Widget')).toBeInTheDocument()
  })

  it('Invoice opens the printable view and triggers the browser print dialog', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    const printSpy = vi.fn()
    vi.stubGlobal('print', printSpy)

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByTitle('Print invoice'))

    expect(await screen.findByText('Invoice')).toBeInTheDocument()
    expect(printSpy).toHaveBeenCalled()
  })

  it('flags a line item with no sku_id as not linked to a product yet', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      line_items: [{
        id: 'li-2', sku_id: null, name: 'Mystery Item', sku_code: 'UNKNOWN-SKU', barcode: '',
        quantity: 1, price_minor_units: 500, subtotal_minor_units: 500, subtotal_tax_minor_units: 0,
        total_minor_units: 500, total_tax_minor_units: 0,
      }],
    }))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Mystery Item')).toBeInTheDocument()
    expect(screen.getByText('Not linked to a product yet')).toBeInTheDocument()
  })

  it('does not flag a line item that has a sku_id', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Blue Widget')).toBeInTheDocument()
    expect(screen.queryByText('Not linked to a product yet')).not.toBeInTheDocument()
  })

  it('shows source and shipping method', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      sales_order: salesOrder({ source: 'woocommerce', shipping_method_title: 'Flat rate' }),
    }))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('woocommerce')).toBeInTheDocument()
    expect(screen.getByText('Flat rate')).toBeInTheDocument()
  })

  it('shows payment_method_title when present', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      sales_order: salesOrder({ payment_method: 'cod', payment_method_title: 'Cash on delivery' }),
    }))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Cash on delivery')).toBeInTheDocument()
  })

  it('falls back to payment_method when payment_method_title is empty (POS orders)', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      sales_order: salesOrder({ payment_method: 'pos', payment_method_title: '' }),
    }))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('pos', { selector: 'span' })).toBeInTheDocument()
  })

  it('hides the Payment row when neither payment_method nor payment_method_title is set', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail({
      sales_order: salesOrder({ payment_method: undefined, payment_method_title: undefined }),
    }))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    expect(screen.queryByText('Payment:')).not.toBeInTheDocument()
  })

  it('Back returns to the list', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByRole('button', { name: /Sales Order/ }))

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.queryByText('Blue Widget')).not.toBeInTheDocument()
  })

  it('shows an error if fetching the detail fails', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockRejectedValue(new Error('not found'))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('not found')).toBeInTheDocument()
  })

  it('changes status via the dropdown and calls updateSalesOrder', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    vi.mocked(omsApi.updateSalesOrder).mockResolvedValue(salesOrder({ status: 'cancelled' }))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.selectOptions(screen.getByDisplayValue('completed'), 'cancelled')

    expect(omsApi.updateSalesOrder).toHaveBeenCalledWith('test-token', 'so-1', { status: 'cancelled' })
  })

  it('deletes the sales order after confirmation and returns to the list', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    vi.mocked(omsApi.deleteSalesOrder).mockResolvedValue(null)
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByTitle('Delete sales order'))

    expect(omsApi.deleteSalesOrder).toHaveBeenCalledWith('test-token', 'so-1')
    expect(await screen.findByRole('button', { name: 'New Sales Order' })).toBeInTheDocument()
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    vi.stubGlobal('confirm', vi.fn(() => false))

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Blue Widget')

    await userEvent.click(screen.getByTitle('Delete sales order'))

    expect(omsApi.deleteSalesOrder).not.toHaveBeenCalled()
  })
})

describe('Create sales order', () => {
  it('opening New Sales Order shows a blank create form', async () => {
    render(<SalesOrders />)
    await userEvent.click(await screen.findByRole('button', { name: 'New Sales Order' }))

    expect(screen.getByText('New Sales Order', { selector: 'h1' })).toBeInTheDocument()
    expect(await screen.findByText('Main Warehouse')).toBeInTheDocument()
  })

  it('adds a line item via the product picker and creates the sales order', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({
      products: [{
        id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Widget Product',
        slug: 'widget-product', description: '', status: 'active', has_variants: false, tags: [],
        created_at: '', updated_at: '',
      }],
      total: 1,
    })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: {
        id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Widget Product',
        slug: 'widget-product', description: '', status: 'active', has_variants: false, tags: [],
        created_at: '', updated_at: '',
      },
      variants: [{
        id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0,
        created_at: '', updated_at: '',
        skus: [{
          id: 'sku-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: 'WID-001',
          price_minor_units: 999, currency_code: 'USD', is_virtual: false, status: 'active',
          created_at: '', updated_at: '',
        }],
      }],
    })
    vi.mocked(omsApi.createSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByRole('button', { name: 'New Sales Order' }))
    await userEvent.selectOptions(screen.getByLabelText('Warehouse'), 'wh-1')

    await userEvent.click(screen.getByRole('button', { name: /Add product/ }))
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Widget')
    await userEvent.click(await screen.findByText('Widget Product'))
    await userEvent.click(await screen.findByText('WID-001'))

    await userEvent.click(screen.getByRole('button', { name: 'Create Sales Order' }))

    await waitFor(() => expect(omsApi.createSalesOrder).toHaveBeenCalledWith('test-token', expect.objectContaining({
      warehouse_id: 'wh-1',
      line_items: [expect.objectContaining({ sku_id: 'sku-1', quantity: 1, price_minor_units: 999 })],
    })))
  })

  it('requires at least one line item', async () => {
    render(<SalesOrders />)
    await userEvent.click(await screen.findByRole('button', { name: 'New Sales Order' }))
    await userEvent.selectOptions(screen.getByLabelText('Warehouse'), 'wh-1')

    await userEvent.click(screen.getByRole('button', { name: 'Create Sales Order' }))

    expect(await screen.findByText('Add at least one line item')).toBeInTheDocument()
    expect(omsApi.createSalesOrder).not.toHaveBeenCalled()
  })

  it('computes per-line and order-level tax from the SKU tax class and tenant defaults', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'USD', timezone: '', date_format: '', language: '',
      prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([
      { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
    ])
    vi.mocked(taxApi.listTaxRates).mockResolvedValue([
      { id: 'rate-1', tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2100, created_at: '', updated_at: '' },
    ])
    vi.mocked(pmsApi.listProducts).mockResolvedValue({
      products: [{
        id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Widget Product',
        slug: 'widget-product', description: '', status: 'active', has_variants: false, tags: [],
        created_at: '', updated_at: '',
      }],
      total: 1,
    })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: {
        id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Widget Product',
        slug: 'widget-product', description: '', status: 'active', has_variants: false, tags: [],
        created_at: '', updated_at: '',
      },
      variants: [{
        id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0,
        created_at: '', updated_at: '',
        skus: [{
          id: 'sku-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: 'WID-001',
          price_minor_units: 999, currency_code: 'USD', is_virtual: false, status: 'active',
          created_at: '', updated_at: '',
        }],
      }],
    })
    vi.mocked(omsApi.createSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByRole('button', { name: 'New Sales Order' }))
    await userEvent.selectOptions(screen.getByLabelText('Warehouse'), 'wh-1')

    await userEvent.click(screen.getByRole('button', { name: /Add product/ }))
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Widget')
    await userEvent.click(await screen.findByText('Widget Product'))
    await userEvent.click(await screen.findByText('WID-001'))

    expect(await screen.findByText(/Tax/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Create Sales Order' }))

    await waitFor(() => expect(omsApi.createSalesOrder).toHaveBeenCalledWith('test-token', expect.objectContaining({
      tax_total_minor_units: 173,
      line_items: [expect.objectContaining({
        sku_id: 'sku-1', quantity: 1, price_minor_units: 999,
        subtotal_minor_units: 999, subtotal_tax_minor_units: 173,
        total_minor_units: 999, total_tax_minor_units: 173,
      })],
    })))
  })

  it('Discard returns to the list without creating', async () => {
    render(<SalesOrders />)
    await userEvent.click(await screen.findByRole('button', { name: 'New Sales Order' }))
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(omsApi.createSalesOrder).not.toHaveBeenCalled()
    expect(await screen.findByRole('button', { name: 'New Sales Order' })).toBeInTheDocument()
  })
})
