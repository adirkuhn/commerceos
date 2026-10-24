import { test, expect, API_BASE_URL } from './fixtures'

// The order page shows each line without VAT, its VAT and with VAT, and a
// refund at what it gives back incl. VAT. The order and its line changes
// are served from fixtures shaped like order #26217 (a Woo order with a
// refunded line and a charged extra) on top of a real order's response.
const ORDER_ID = process.env.E2E_SALES_ORDER_ID ?? ''
const API_ORIGIN = new URL(API_BASE_URL).origin

const LINES = [
  { id: 'li-tapioca', sku_id: 'sku-tap', name: 'Tapioca', sku_code: 'TAP-1', barcode: '', quantity: 2, price_minor_units: 458,
    subtotal_minor_units: 916, subtotal_tax_minor_units: 82, total_minor_units: 916, total_tax_minor_units: 82,
    total_includes_tax: false, total_excl_tax_minor_units: 916, total_incl_tax_minor_units: 998 },
  { id: 'li-caldo', sku_id: 'sku-cal', name: 'Caldo de Cana', sku_code: 'CAL-1', barcode: '', quantity: 3, price_minor_units: 458,
    subtotal_minor_units: 1373, subtotal_tax_minor_units: 124, total_minor_units: 1373, total_tax_minor_units: 124,
    total_includes_tax: false, total_excl_tax_minor_units: 1373, total_incl_tax_minor_units: 1497, refunded_quantity: 3, refunded_amount_minor_units: 1497 },
  { id: 'li-canj', sku_id: 'sku-canj', name: 'Canjiquinha', sku_code: 'CAN-1', barcode: '', quantity: 2, price_minor_units: 199,
    subtotal_minor_units: 398, subtotal_tax_minor_units: 33, total_minor_units: 398, total_tax_minor_units: 33,
    total_includes_tax: true, total_excl_tax_minor_units: 365, total_incl_tax_minor_units: 398 },
]

const CHANGES = [
  { id: 'c-refund', step: 'packing', action: 'refund', line_item_id: 'li-caldo', sku_id: 'sku-cal', sku_code: 'CAL-1', product_name: 'Caldo de Cana',
    variant_name: '', quantity: 3, return_to_stock: false, amount_minor_units: 1497, reason: 'sem estoque', by: 'user-1', at: '2026-10-24T10:49:00Z' },
  { id: 'c-extra', step: 'packing', action: 'new_item', line_item_id: 'li-canj', sku_id: 'sku-canj', sku_code: 'CAN-1', product_name: 'Canjiquinha',
    variant_name: '', quantity: 2, return_to_stock: false, price_minor_units: 199, amount_minor_units: 398, reason: 'Canjiquinha', by: 'user-1', at: '2026-10-24T10:50:00Z' },
]

test('order lines show excl. VAT, VAT and incl. VAT', async ({ page }) => {
  test.skip(ORDER_ID === '', 'needs E2E_SALES_ORDER_ID (any sales order id of the tenant)')
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith(`/sales-orders/${ORDER_ID}`), async route => {
    const response = await route.fetch()
    const body = await response.json()
    body.data.line_items = LINES
    body.data.sales_order.currency_code = 'EUR'
    await route.fulfill({ response, json: body, headers: { ...response.headers(), 'access-control-allow-origin': '*' } })
  })
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith(`/sales-orders/${ORDER_ID}/line-changes`), route => route.fulfill({
    status: 200, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' },
    body: JSON.stringify({ data: { changes: CHANGES }, error: null }),
  }))

  await page.addInitScript(() => localStorage.setItem('commerceos.amounts', 'visible'))
  await page.goto(`/sales-orders/${ORDER_ID}`)
  const caldo = page.locator('tr', { hasText: 'Caldo de Cana' })
  await expect(caldo.getByRole('cell').nth(2)).toHaveText(/13[.,]73/)
  await expect(caldo.getByRole('cell').nth(3)).toHaveText(/1[.,]24/)
  await expect(caldo.getByRole('cell').nth(4)).toHaveText(/14[.,]97/)
  await expect(caldo).toContainText(/Refunded 3 \(€\s?14[.,]97\)/)
  await expect(page.locator('thead').first()).toContainText(/Excl. VAT\s*VAT\s*Incl. VAT/)
  await page.screenshot({ path: 'e2e-screenshots/sales-order-line-vat.png', fullPage: true })
  await page.setViewportSize({ width: 390, height: 844 })
  await expect(caldo).toBeVisible()
  await page.screenshot({ path: 'e2e-screenshots/sales-order-line-vat-mobile.png', fullPage: true })
})
