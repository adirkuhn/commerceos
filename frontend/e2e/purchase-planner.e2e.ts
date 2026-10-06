import { test, expect, api, API_BASE_URL } from './fixtures'

interface Supplier { id: string; name: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string; sku_code: string }
interface PurchaseOrderDetail {
  purchase_order: { id: string; supplier_id: string | null; status: string; currency_code: string }
  line_items: { sku_id: string | null; quantity: number; units_per_purchased_item: number }[]
}

// The planner's numbers come from sales history, and sales can only be
// created through the licensed POS API -- so this spec serves the
// recommendations response itself (a plan for a real supplier and SKU it
// creates) and lets everything else hit the real backend: creating the
// draft PO from the plan, the procurement editor it opens, and the
// season-events list.
test('turn a supplier recommendation into a draft PO and open it', async ({ page, request, token }) => {
  const stamp = Date.now()
  const supplier = await api<Supplier>(request, token, '/suppliers', {
    method: 'POST', body: { name: `E2E Bauducco ${stamp}`, status: 'active', lead_time_days: 21 },
  })
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name: `E2E Panettone ${stamp}`, status: 'active' } })
  let purchaseOrderId = ''
  try {
    const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
    const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
      method: 'POST', body: { sku_code: `E2E-PAN-${stamp}`, price_minor_units: 1299, currency_code: 'EUR', status: 'active' },
    })

    await page.route(`${new URL(API_BASE_URL).origin}/**/procurement/recommendations?*`, route => route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' },
      body: JSON.stringify({
        data: {
          as_of: '2026-10-06', coverage_days: 30, safety_days: 7,
          suppliers: [{
            supplier_id: supplier.id, supplier_name: supplier.name, lead_time_days: 21, lead_time_assumed: false,
            currency_code: 'EUR', estimated_cost_minor_units: 77400, earliest_order_by_date: '2026-10-23',
            lines: [{
              sku_id: sku.id, sku_code: sku.sku_code, product_name: `E2E Panettone ${stamp}`, variant_name: 'Default',
              is_weight_pool: false, on_hand: 12, on_order: { sent: 0, draft: 0 },
              forecast: { units: 968, method: 'seasonal', growth_factor: 1.18, days: 51, recent_units: 118, last_year_units: 820 },
              daily_demand: 19, safety_stock: 133, needed_units: 1031, purchase_quantity: 172, units_per_purchased_item: 6,
              order_units: 1032, unit_cost_minor_units: 450, estimated_cost_minor_units: 77400, currency_code: 'EUR',
              order_by_date: '2026-10-23',
            }],
          }],
        },
        error: null,
      }),
    }))

    await page.goto('/purchase-planner')
    await expect(page.getByRole('heading', { name: supplier.name })).toBeVisible()
    await expect(page.getByText('Last year 820 × 1.18')).toBeVisible()
    const quantity = page.getByLabel(`Quantity for E2E Panettone ${stamp}`)
    await expect(quantity).toHaveValue('172')
    await quantity.fill('150')
    await page.getByRole('button', { name: 'Create draft PO' }).click()

    await expect(page).toHaveURL(/\/procurement\/[0-9a-f-]{36}$/)
    purchaseOrderId = page.url().split('/').pop() ?? ''
    const detail = await api<PurchaseOrderDetail>(request, token, `/procurement/purchase-orders/${purchaseOrderId}`, { method: 'GET' })
    expect(detail.purchase_order).toMatchObject({ supplier_id: supplier.id, status: 'draft' })
    expect(detail.line_items).toHaveLength(1)
    // Never bought from this supplier before: pack size 1, no price -- the
    // plan's box of 6 and price are not trusted from the browser.
    expect(detail.line_items[0]).toMatchObject({ sku_id: sku.id, quantity: 150, units_per_purchased_item: 1 })
    await expect(page.getByText(`E2E Panettone ${stamp}`).first()).toBeVisible()

    // Back returns to the plan.
    await page.goBack()
    await expect(page).toHaveURL(/\/purchase-planner/)
  } finally {
    if (purchaseOrderId) await api(request, token, `/procurement/purchase-orders/${purchaseOrderId}`, { method: 'DELETE' })
    await api(request, token, `/pms/products/${product.id}`, { method: 'DELETE' })
    await api(request, token, `/suppliers/${supplier.id}`, { method: 'DELETE' })
  }
})

test('season plan offers the upcoming holidays and keeps the choice in the URL', async ({ page }) => {
  await page.goto('/purchase-planner')
  await page.getByRole('tab', { name: 'Season plan' }).click()
  await expect(page).toHaveURL(/tab=season/)

  const season = page.getByLabel('Season')
  await expect(season.locator('option', { hasText: /^Christmas \(25\/12\/\d{4}\)$/ })).toHaveCount(1)
  await expect(season.locator('option', { hasText: /^Easter \(/ })).toHaveCount(1)
  await season.selectOption('christmas')
  await expect(page).toHaveURL(/season=christmas/)
  await expect(page.getByText(/^Christmas \d{4}$/)).toBeVisible()

  await season.selectOption('custom')
  await expect(page.getByLabel('From')).toBeVisible()
  await expect(page.getByLabel('To')).toBeVisible()
})
