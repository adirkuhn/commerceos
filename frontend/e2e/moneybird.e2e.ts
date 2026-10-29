import { test, expect, API_BASE_URL } from './fixtures'

// Moneybird, through the admin: its settings page (administration, ledger,
// VAT and payment-method mappings, auto sync) and the card that sends an
// order. Moneybird itself and the order are served from fixtures; every
// write is caught, not sent.
const API_ORIGIN = new URL(API_BASE_URL).origin
const CONN_ID = '3b9f6c1e-2a4d-4e8f-9b7a-5c6d7e8f9a01'
const ORDER_ID = '8c2e4f6a-1b3d-4c5e-8f7a-9b0c1d2e3f40'
const json = (data: unknown, status = 200) => ({ status, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' }, body: JSON.stringify({ data, error: null }) })
const fail = (status: number, code: string) => ({ status, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' }, body: JSON.stringify({ data: null, error: { code, message: code } }) })
const isApi = (path: string) => (url: URL) => url.origin === API_ORIGIN && url.pathname.endsWith(path)

const SETUP = {
  settings: {
    connection_id: CONN_ID, administration_id: 'adm-1', administration_name: 'Cantinho Brasileiro', default_ledger_account_id: 'led-1',
    auto_sync: false, tax_rates: [{ rate_basis_points: 2100, moneybird_tax_rate_id: 'tr-21' }],
    payment_methods: [{ method: 'bacs', moneybird_payment_method: 'bank_transfer' }],
  },
  administrations: [{ id: 'adm-1', name: 'Cantinho Brasileiro', currency: 'EUR' }],
  tax_rates: [{ id: 'tr-21', name: '21% btw', percentage: '21.0' }, { id: 'tr-9', name: '9% btw', percentage: '9.0' }, { id: 'tr-0', name: '0% btw', percentage: '0.0' }],
  ledger_accounts: [{ id: 'led-1', name: 'Omzet' }],
  payment_methods: ['cash', 'credit_card', 'debit_card', 'bank_transfer', 'ideal', 'paypal', 'pin'],
}

const ORDER = {
  sales_order: {
    id: ORDER_ID, tenant_id: 't', source: 'woocommerce', external_order_id: '41235', number: '41235', status: 'processing', payment_status: 'paid',
    currency_code: 'EUR', warehouse_id: 'wh-e2e', customer_note: '', discount_total_minor_units: 0, shipping_total_minor_units: 495,
    tax_total_minor_units: 162, total_minor_units: 1297, occurred_at: '2026-10-28T09:00:00Z', created_at: '2026-10-28T09:00:00Z', fulfilled: false,
    billing: { first_name: 'Ana', last_name: 'Souza', company: 'Mercado Lusitano B.V.', address_1: 'Kerkstraat 10', postcode: '1017GB', city: 'Amsterdam', country: 'NL' },
    shipping: { first_name: 'Ana', last_name: 'Souza', address_1: 'Kerkstraat 10', postcode: '1017GB', city: 'Amsterdam', country: 'NL' },
  },
  line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Farofa Yoki 250g', sku_code: 'FAR-250', barcode: '', quantity: 2, price_minor_units: 320,
    subtotal_minor_units: 640, subtotal_tax_minor_units: 58, total_minor_units: 640, total_tax_minor_units: 58, total_includes_tax: false,
    total_excl_tax_minor_units: 640, total_incl_tax_minor_units: 698, stock_status: 'decremented' }],
  payments: [],
}

test('Moneybird settings: map a VAT rate and turn auto sync on', async ({ page }) => {
  const patches: unknown[] = []
  await page.route(isApi('/integrations/connections'), route => route.fulfill(json([
    { id: CONN_ID, provider: 'moneybird', store_url: '', status: 'connected', created_at: '2026-10-29T08:00:00Z' },
  ])))
  let current = structuredClone(SETUP)
  await page.route(isApi('/integrations/moneybird'), route => route.fulfill(json(current)))
  await page.route(isApi('/integrations/moneybird/settings'), async route => {
    const patch = route.request().postDataJSON()
    patches.push(patch)
    current = { ...current, settings: { ...current.settings, ...patch } }
    await route.fulfill(json(current.settings))
  })

  await page.goto(`/integrations/${CONN_ID}`)
  await expect(page.getByRole('heading', { name: 'Moneybird' })).toBeVisible()
  await expect(page.getByRole('combobox', { name: 'Administration' })).toHaveValue('adm-1')
  await page.screenshot({ path: 'e2e-screenshots/moneybird-settings.png', fullPage: true })

  const vat = page.getByRole('region', { name: 'VAT rates' })
  await expect(vat.getByLabel('VAT 21%')).toHaveValue('tr-21')
  await vat.getByLabel('Add VAT rate (%)').fill('9')
  await vat.getByRole('button', { name: 'Add' }).click()
  await vat.getByLabel('VAT 9%').selectOption('tr-9')
  await vat.getByRole('button', { name: 'Save' }).click()
  await expect(vat.getByRole('status')).toHaveText('Saved')

  await page.getByLabel('Send automatically').check()
  await expect.poll(() => patches.length).toBe(2)
  expect(patches[0]).toEqual({ tax_rates: [
    { rate_basis_points: 2100, moneybird_tax_rate_id: 'tr-21' },
    { rate_basis_points: 900, moneybird_tax_rate_id: 'tr-9' },
  ] })
  expect(patches[1]).toEqual({ auto_sync: true })

  await page.setViewportSize({ width: 390, height: 844 })
  await page.screenshot({ path: 'e2e-screenshots/moneybird-settings-mobile.png', fullPage: true })
})

test.describe('Moneybird card on a sales order', () => {
  test.beforeEach(async ({ page }) => {
    const order = (path: string) => isApi(`/sales-orders/${ORDER_ID}${path}`)
    await page.route(url => order('')(url) && !url.pathname.includes('/moneybird/'), route => route.fulfill(json(ORDER)))
    await page.route(order('/activities'), route => route.fulfill(json({ activities: [] })))
    await page.route(order('/line-changes'), route => route.fulfill(json({ changes: [] })))
    await page.route(order('/bol-shipment-confirmations'), route => route.fulfill(json([])))
    await page.route(url => url.origin === API_ORIGIN && url.pathname.includes(`/shipping/sales-orders/${ORDER_ID}`), route => route.fulfill(json([])))
    await page.route(isApi('/inventory/warehouses'), route => route.fulfill(json([{ id: 'wh-e2e', name: 'Hoofddorp', created_at: '', updated_at: '' }])))
    await page.route(isApi(`/integrations/moneybird/sales-orders/${ORDER_ID}`), route =>
      route.fulfill(json({ document: null, administration_id: 'adm-1', auto_sync: false })))
    await page.addInitScript(() => localStorage.setItem('commerceos.amounts', 'visible'))
  })

  test('books a paid order and links to the invoice', async ({ page }) => {
    await page.route(isApi(`/integrations/moneybird/sales-orders/${ORDER_ID}/send`), route => route.fulfill(json({
      source_type: 'sales_order', source_id: ORDER_ID, moneybird_id: '4567', reference: '41235', status: 'synced',
      payment_registered: true, error: '', synced_at: '2026-10-29T10:00:00Z',
    })))

    await page.goto(`/sales-orders/${ORDER_ID}`)
    const card = page.getByRole('region', { name: 'Moneybird' })
    await expect(card.getByText('Not sent to Moneybird yet.')).toBeVisible()
    await card.getByRole('button', { name: 'Send to Moneybird' }).click()

    await expect(card.getByText('Booked in Moneybird, payment registered.')).toBeVisible()
    await expect(card.getByRole('link', { name: 'Open in Moneybird' })).toHaveAttribute('href', 'https://moneybird.com/adm-1/external_sales_invoices/4567')
    await expect(card.getByRole('button')).toHaveCount(0)
    await card.scrollIntoViewIfNeeded()
    await page.screenshot({ path: 'e2e-screenshots/moneybird-order-booked.png', fullPage: true })
  })

  test('an order another service already booked is refused, with the reason', async ({ page }) => {
    let sends = 0
    await page.route(isApi(`/integrations/moneybird/sales-orders/${ORDER_ID}/send`), async route => {
      sends++
      await route.fulfill(fail(409, 'MONEYBIRD_DUPLICATE_REFERENCE'))
    })

    await page.goto(`/sales-orders/${ORDER_ID}`)
    const card = page.getByRole('region', { name: 'Moneybird' })
    await card.getByRole('button', { name: 'Send to Moneybird' }).click()

    await expect(card.getByRole('alert')).toHaveText('Moneybird already has an invoice with this number. It was not booked again.')
    expect(sends).toBe(1)
    await card.scrollIntoViewIfNeeded()
    await page.screenshot({ path: 'e2e-screenshots/moneybird-order-duplicate.png', fullPage: true })
  })
})
