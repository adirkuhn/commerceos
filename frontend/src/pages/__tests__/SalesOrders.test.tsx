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
  createSalesOrder: vi.fn(),
  updateSalesOrder: vi.fn(),
  deleteSalesOrder: vi.fn(),
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
      id: 'li-1', sku_id: 'sku-1', name: 'Blue Widget', sku_code: 'WID-BLU',
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
    prices_include_tax: true, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
  })
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listTaxRates).mockResolvedValue([])
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

describe('Sales order detail page', () => {
  it('opens the detail page and shows line items when clicking a row', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    render(<SalesOrders />)
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Blue Widget')).toBeInTheDocument()
    expect(screen.getByText('WID-BLU')).toBeInTheDocument()
    expect(omsApi.getSalesOrder).toHaveBeenCalledWith('test-token', 'so-1')
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
      prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
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
