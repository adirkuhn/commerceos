import { test, expect, api, API_BASE_URL } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

interface Warehouse { id: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }
interface Batch { quantity: number }
interface ResolvedBarcode { sku: { id: string } }

async function productWithStock(
  request: APIRequestContext, token: string, warehouseId: string, name: string, sku: Record<string, unknown>, qty: number,
): Promise<{ productId: string; skuId: string }> {
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
  const created = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
    method: 'POST', body: { price_minor_units: 100, currency_code: 'EUR', status: 'active', ...sku },
  })
  await api(request, token, `/inventory/skus/${created.id}/movements`, {
    method: 'POST', body: { warehouse_id: warehouseId, type: 'receipt', quantity_delta: qty },
  })
  return { productId: product.id, skuId: created.id }
}

// A product booked in as new when it already existed is merged into the
// existing one: stock moves, the duplicate's barcode keeps scanning, the
// duplicate is gone and the editor lands on the existing product.
test('merge a duplicate product into the existing one', async ({ page, request, token }) => {
  const probe = await request.post(`${API_BASE_URL}/pms/products/none/merge-preview`, {
    headers: { Authorization: `Bearer ${token}` }, data: { target_product_id: 'none' },
  })
  test.skip(probe.status() === 404 && !(await probe.text()).includes('product'), 'backend predates product merge -- rebuild the local stack')

  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const existingName = `E2E Merge ${stamp} existing`
  const duplicateBarcode = `20${String(stamp).slice(-11)}`
  const existing = await productWithStock(request, token, warehouse.id, existingName, { sku_code: `E2E-MX-${stamp}` }, 6)
  const duplicate = await productWithStock(request, token, warehouse.id, `E2E Merge ${stamp} duplicate`,
    { sku_code: `E2E-MD-${stamp}`, barcode: duplicateBarcode }, 4)
  try {
    await page.goto(`/products/${duplicate.productId}`)
    await page.getByRole('button', { name: /Merge into/ }).click()
    const dialog = page.getByRole('dialog', { name: /Merge .* into another product/ })
    await dialog.getByLabel('The existing product to keep').fill(existingName)
    await dialog.getByRole('button', { name: existingName }).click()

    await expect(dialog.getByLabel('After the merge')).toContainText('10')
    const mergeButton = dialog.getByRole('button', { name: 'Merge permanently' })
    await expect(mergeButton).toBeDisabled()
    await dialog.getByLabel(/to confirm/).fill(existingName)
    await mergeButton.click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Merge now' }).click()

    await expect(page).toHaveURL(new RegExp(`/products/${existing.productId}`))

    const batches = await api<Batch[]>(request, token, `/inventory/skus/${existing.skuId}/batches`, { method: 'GET' })
    expect(batches.reduce((sum, b) => sum + b.quantity, 0)).toBe(10)
    const scanned = await api<ResolvedBarcode>(request, token, `/pms/skus/barcode-lookup/${duplicateBarcode}`, { method: 'GET' })
    expect(scanned.sku.id).toBe(existing.skuId)
    const gone = await request.get(`${API_BASE_URL}/pms/products/${duplicate.productId}`, { headers: { Authorization: `Bearer ${token}` } })
    expect(gone.status()).toBe(404)
  } finally {
    for (const id of [duplicate.productId, existing.productId]) {
      await api(request, token, `/pms/products/${id}`, { method: 'DELETE' }).catch(() => undefined)
    }
  }
})
