import { test, expect, API_BASE_URL } from './fixtures'

// A product with no VAT rate configured (no default VAT class, or no rate
// for the tax country) is named in a warning on the order form instead of
// going in at 0% unnoticed -- how manual order CO-1002 reached Moneybird at
// 0%. Order, price list and tax setup are served from fixtures.
const API_ORIGIN = new URL(API_BASE_URL).origin
const ORDER_ID = '7d3f2a10-5b8e-4c1a-9f0e-2b6c8d4e1a02'
const json = (data: unknown) => ({ status: 200, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' }, body: JSON.stringify({ data, error: null }) })

const ORDER = {
  sales_order: {
    id: ORDER_ID, tenant_id: 't', source: 'manual', external_order_id: ORDER_ID, number: 'CO-1001', status: 'processing', payment_status: 'paid',
    currency_code: 'EUR', warehouse_id: 'wh-e2e', customer_note: '', discount_total_minor_units: 0, shipping_total_minor_units: 799,
    tax_total_minor_units: 0, total_minor_units: 5882, occurred_at: '2026-10-26T11:40:00Z', created_at: '2026-10-26T11:40:00Z', fulfilled: false,
    billing: { first_name: 'Kylian', last_name: 'Klockenbrink', company: 'Claus Hoofddorp B.V.', address_1: 'Bosweg 19', postcode: '2131LX', city: 'Hoofddorp', country: 'NL' },
    shipping: { first_name: 'Kylian', last_name: 'Klockenbrink', company: 'Claus Hoofddorp B.V.', address_1: 'Bosweg 19', postcode: '2131LX', city: 'Hoofddorp', country: 'NL' },
  },
  line_items: [{ id: 'li-mate', sku_id: 'sku-mate', name: 'Chá Mate Leão - Granel - 100g', sku_code: '7891098000026', barcode: '7891098000026',
    quantity: 17, price_minor_units: 299, subtotal_minor_units: 5083, subtotal_tax_minor_units: 0, total_minor_units: 5083, total_tax_minor_units: 0,
    total_includes_tax: true, total_excl_tax_minor_units: 5083, total_incl_tax_minor_units: 5083, stock_status: 'decremented' }],
  payments: [],
}
const MATE_ROW = {
  product_id: 'p-mate', product_name: 'Chá Mate Leão - Granel - 100g', variant_name: '', sku_id: 'sku-mate', sku_code: '7891098000026', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 299, available_quantity: 40, tax_rate_basis_points: 0, tax_rate_configured: false, target_markup_basis_points: 0,
}

test.beforeEach(async ({ page }) => {
  const order = (path: string) => (url: URL) => url.origin === API_ORIGIN && url.pathname.endsWith(`/sales-orders/${ORDER_ID}${path}`)
  await page.route(order(''), route => route.request().method() === 'PUT' ? route.fallback() : route.fulfill(json(ORDER)))
  await page.route(order('/activities'), route => route.fulfill(json({ activities: [] })))
  await page.route(order('/line-changes'), route => route.fulfill(json({ changes: [] })))
  await page.route(order('/bol-shipment-confirmations'), route => route.fulfill(json([])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.includes(`/shipping/sales-orders/${ORDER_ID}`), route => route.fulfill(json([])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/inventory/warehouses'), route => route.fulfill(json([{ id: 'wh-e2e', name: 'Hoofddorp', created_at: '', updated_at: '' }])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/pms/price-list'), route => route.fulfill({
    status: 200, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' },
    body: JSON.stringify({ data: [MATE_ROW], error: null, meta: { total: 1, prices_include_tax: true } }),
  }))
  await page.addInitScript(() => localStorage.setItem('commerceos.amounts', 'visible'))
})

test('the order form names products without a VAT rate', async ({ page }) => {
  // Tax setup is complete for the default class; this product's own class has no rate.
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/tax/classes'), route =>
    route.fulfill(json([{ id: 'std', name: 'Standard', is_default: true, created_at: '', updated_at: '' }])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/tax/rates'), route =>
    route.fulfill(json([{ id: 'r1', tax_class_id: 'std', country: 'NL', rate_basis_points: 2100, created_at: '', updated_at: '' }])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/settings/tenant'), route =>
    route.fulfill(json({ tenant_id: 't', prices_include_tax: true, default_tax_country: 'NL' })))

  await page.goto(`/sales-orders/${ORDER_ID}`)
  await page.getByRole('link', { name: 'Edit' }).click()

  await expect(page.getByText('No VAT rate for Chá Mate Leão - Granel - 100g: these lines are at 0%.')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Set it in Settings → Tax' })).toHaveAttribute('href', '/settings/tax')
  await page.screenshot({ path: 'e2e-screenshots/sales-order-missing-vat.png', fullPage: true })
})
