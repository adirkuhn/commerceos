// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import i18next from 'i18next'
import UnmatchedSkus from '../UnmatchedSkus'
import { renderAt } from '../../test/renderAt'
import type { SalesOrder, SalesOrderDetail } from '../../oms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  getSalesOrder: vi.fn(),
  updateSalesOrder: vi.fn(),
  deleteSalesOrder: vi.fn(),
  linkLineItemSku: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({ listWarehouses: vi.fn() }))
vi.mock('../../pms/api', () => ({ listProducts: vi.fn(), getProduct: vi.fn() }))
vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn(), updateTenantSettings: vi.fn() }))
vi.mock('../../tax/api', () => ({
  listTaxClasses: vi.fn(), createTaxClass: vi.fn(), updateTaxClass: vi.fn(), deleteTaxClass: vi.fn(),
  listTaxRates: vi.fn(), upsertTaxRate: vi.fn(), deleteTaxRate: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as pmsApi from '../../pms/api'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    needs_sku_review: true,
    ...overrides,
  }
}

function detail(overrides: Partial<SalesOrderDetail> = {}): SalesOrderDetail {
  return {
    sales_order: salesOrder(),
    line_items: [{
      id: 'li-1', sku_id: null, name: 'Mystery Item', sku_code: 'UNKNOWN-SKU', barcode: '',
      quantity: 1, stock_status: 'pending', price_minor_units: 1000, subtotal_minor_units: 1000,
      subtotal_tax_minor_units: 0, total_minor_units: 1000, total_tax_minor_units: 0,
    }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
})

describe('UnmatchedSkus', () => {
  it('shows an empty state when there are no unmatched SKUs', async () => {
    renderAt(<UnmatchedSkus />, '/unmatched-skus')

    expect(await screen.findByText("No unmatched SKUs — everything's linked")).toBeInTheDocument()
  })

  it('lists orders flagged for SKU review, requesting the needsSkuReview filter', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    renderAt(<UnmatchedSkus />, '/unmatched-skus')

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', expect.objectContaining({ needsSkuReview: true }))
  })

  it('expands an order to show its unmatched line item', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())

    renderAt(<UnmatchedSkus />, '/unmatched-skus')
    await userEvent.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Mystery Item')).toBeInTheDocument()
    expect(screen.getByText('Woo SKU: UNKNOWN-SKU · Qty 1')).toBeInTheDocument()
  })

  it('links a line item to a product/SKU via the picker', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail())
    vi.mocked(pmsApi.listProducts).mockResolvedValue({
      products: [{ id: 'product-1', name: 'Blue Widget' } as never], total: 1,
    })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      variants: [{ skus: [{ id: 'sku-1', sku_code: 'WID-BLU', price_minor_units: 2000, currency_code: 'USD' }] }],
    } as never)
    vi.mocked(omsApi.linkLineItemSku).mockResolvedValue(salesOrder({ needs_sku_review: false }))

    renderAt(<UnmatchedSkus />, '/unmatched-skus')
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Mystery Item')

    await userEvent.click(screen.getByRole('button', { name: 'Link to product' }))
    await userEvent.click(await screen.findByText('Blue Widget'))
    await userEvent.click(await screen.findByText('WID-BLU'))

    expect(omsApi.linkLineItemSku).toHaveBeenCalledWith('test-token', 'so-1', 'li-1', { sku_id: 'sku-1' })
  })
})

describe('UnmatchedSkus URLs', () => {
  it('requests the page in ?page=', async () => {
    renderAt(<UnmatchedSkus />, '/unmatched-skus', '/unmatched-skus?page=3')
    await waitFor(() => expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', expect.objectContaining({ page: 3 })))
  })
})

describe('in Brazilian Portuguese', () => {
  afterEach(async () => { await i18next.changeLanguage('en') })

  it('shows the empty state in pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    renderAt(<UnmatchedSkus />, '/unmatched-skus')

    expect(await screen.findByText('Nenhum SKU sem vínculo — está tudo vinculado')).toBeInTheDocument()
  })
})
