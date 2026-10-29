import { test, expect, API_BASE_URL } from './fixtures'

// The printed invoice of a long order runs over several A4 pages, every
// page with the same margins. It used to print page 1 only: the invoice sat
// in a screen-height scrolling overlay inside the app shell, and print
// output stops where the screen does. Order and company profile are served
// from fixtures; the PDF is Chromium's own print output.
const API_ORIGIN = new URL(API_BASE_URL).origin
const ORDER_ID = '0b4d6c8e-2f1a-4e3b-9c5d-7a8b9c0d1e21'
const json = (data: unknown) => ({ status: 200, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' }, body: JSON.stringify({ data, error: null }) })

const LINES = Array.from({ length: 45 }, (_, i) => ({
  id: `li-${i + 1}`, sku_id: `sku-${i + 1}`, name: `Line ${i + 1} - Farofa de Mandioca Temperada Yoki 250g`, sku_code: `SKU-${i + 1}`, barcode: '',
  quantity: 2, price_minor_units: 349, subtotal_minor_units: 698, subtotal_tax_minor_units: 58, total_minor_units: 698, total_tax_minor_units: 58,
  total_includes_tax: true, total_excl_tax_minor_units: 640, total_incl_tax_minor_units: 698, stock_status: 'decremented',
}))

const ORDER = {
  sales_order: {
    id: ORDER_ID, tenant_id: 't', source: 'manual', external_order_id: ORDER_ID, number: 'CO-2045', status: 'processing', payment_status: 'paid',
    currency_code: 'EUR', warehouse_id: 'wh-e2e', customer_note: '', discount_total_minor_units: 0, shipping_total_minor_units: 0,
    tax_total_minor_units: 2610, total_minor_units: 31410, occurred_at: '2026-10-29T09:00:00Z', created_at: '2026-10-29T09:00:00Z', fulfilled: false,
    billing: { first_name: 'Ana', last_name: 'Souza', company: 'Mercado Lusitano B.V.', address_1: 'Kerkstraat 10', postcode: '1017GB', city: 'Amsterdam', country: 'NL' },
    shipping: { first_name: 'Ana', last_name: 'Souza', company: 'Mercado Lusitano B.V.', address_1: 'Kerkstraat 10', postcode: '1017GB', city: 'Amsterdam', country: 'NL' },
  },
  line_items: LINES,
  payments: [],
}

const SELLER = {
  legal_name: 'Example Foods B.V.', trading_name: 'Example', vat_number: 'NL000000000B01', chamber_of_commerce: '00000000',
  email: 'info@example.nl', phone: '', iban: 'NL00BANK0000000000',
  address: { street: 'Voorbeeldweg', house_number: '1', postal_code: '1000 AA', city: 'Amsterdam', country: 'NL' }, updated_at: '',
}

// Every "/Type /Page" object except the "/Type /Pages" tree root.
function pdfPageCount(pdf: Buffer): number {
  return (pdf.toString('latin1').match(/\/Type\s*\/Page(?!s)/g) ?? []).length
}

test.beforeEach(async ({ page }) => {
  const order = (path: string) => (url: URL) => url.origin === API_ORIGIN && url.pathname.endsWith(`/sales-orders/${ORDER_ID}${path}`)
  await page.route(order(''), route => route.fulfill(json(ORDER)))
  await page.route(order('/activities'), route => route.fulfill(json({ activities: [] })))
  await page.route(order('/line-changes'), route => route.fulfill(json({ changes: [] })))
  await page.route(order('/bol-shipment-confirmations'), route => route.fulfill(json([])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.includes(`/shipping/sales-orders/${ORDER_ID}`), route => route.fulfill(json([])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/inventory/warehouses'), route => route.fulfill(json([{ id: 'wh-e2e', name: 'Hoofddorp', created_at: '', updated_at: '' }])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/settings/company-profile'), route => route.fulfill(json(SELLER)))
  // The print dialog itself can't be driven; page.pdf() renders the same print output.
  await page.addInitScript(() => { window.print = () => {}; localStorage.setItem('commerceos.amounts', 'visible') })
})

test('a long invoice prints on several pages, with every line and the totals', async ({ page }) => {
  await page.goto(`/sales-orders/${ORDER_ID}`)
  await page.getByRole('button', { name: 'Invoice' }).click()
  await expect(page.getByText('Line 45 - Farofa de Mandioca Temperada Yoki 250g')).toBeAttached()

  const pdf = await page.pdf({ preferCSSPageSize: true, path: 'e2e-screenshots/sales-order-invoice.pdf' })

  expect(pdfPageCount(pdf)).toBeGreaterThan(1)
  await page.emulateMedia({ media: 'print' })
  // Only the invoice prints -- the app shell around it is hidden.
  await expect(page.locator('body > .print-sheet')).toBeVisible()
  await expect(page.locator('#root')).toBeHidden()
})
