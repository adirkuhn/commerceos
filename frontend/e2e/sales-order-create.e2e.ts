import { test, expect, api } from './fixtures'

// These specs read real figures: show amounts (hidden by default, remembered).
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('commerceos.amounts', 'visible'))
})

interface Warehouse { id: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }

// The new order form: a customer created on the spot (no email), a product
// added twice (one line, qty 2), shipping and an extra fee in the total.
test('sales orders: new order with a new customer, shipping and a fee', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')
  const stamp = Date.now()
  const name = `E2E Order ${stamp}`
  const code = `E2E-SO-${stamp}`
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
  const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
    method: 'POST', body: { sku_code: code, price_minor_units: 1000, currency_code: 'EUR', status: 'active' },
  })
  await api(request, token, `/inventory/skus/${sku.id}/movements`, {
    method: 'POST', body: { warehouse_id: warehouse.id, type: 'receipt', quantity_delta: 10 },
  })

  await page.goto('/sales-orders/new')
  await page.getByRole('button', { name: 'New customer' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('First name').fill(`E2E Bruno ${stamp}`)
  await dialog.getByLabel('City').fill('Delft')
  await dialog.getByRole('button', { name: 'Create customer' }).click()
  await expect(page.locator('#billing-city')).toHaveValue('Delft')

  await page.locator('#so-warehouse').selectOption(warehouse.id)
  await page.getByRole('textbox', { name: /search products/i }).fill(code)
  const add = page.getByRole('button', { name: new RegExp(`add .*${code}`, 'i') })
  await add.click()
  await add.click()
  await expect(page.getByRole('spinbutton', { name: `Qty ${code}` })).toHaveValue('2')

  await page.getByRole('textbox', { name: 'Shipping' }).fill('5')
  await page.getByRole('textbox', { name: 'Shipping' }).press('Enter')
  await page.getByRole('textbox', { name: 'Extra fee' }).fill('Pallet')
  await page.getByRole('textbox', { name: 'Fee amount' }).fill('10')
  await page.getByRole('textbox', { name: 'Fee amount' }).press('Enter')
  await expect(page.getByTestId('order-total')).toContainText('35.00')

  await page.getByRole('button', { name: 'Create sales order' }).click()
  await expect(page.getByText('Pallet')).toBeVisible()
})
