import { test, expect, api } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

interface Warehouse { id: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }

async function createStockedProduct(request: APIRequestContext, token: string, warehouseId: string, name: string, code: string, quantity: number) {
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
  const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
    method: 'POST', body: { sku_code: code, price_minor_units: 450, currency_code: 'EUR', status: 'active' },
  })
  if (quantity !== 0) {
    await api(request, token, `/inventory/skus/${sku.id}/movements`, {
      method: 'POST', body: { warehouse_id: warehouseId, type: 'receipt', quantity_delta: quantity },
    })
  }
  return product.id
}

// The Low stock / Out of stock views are filtered by the server: page 1
// is a full page of matches (not one page of all products filtered in the
// browser), and a product's own low stock line overrides the shop's.
test('inventory: low and out of stock are filtered server-side, product line overrides the shop', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const prefix = `E2E Low stock ${stamp}`
  const ids: string[] = []
  try {
    // 22 products with 1 piece left: more than one page of low stock.
    for (let i = 0; i < 22; i++) {
      ids.push(await createStockedProduct(request, token, warehouse.id, `${prefix} low ${String(i).padStart(2, '0')}`, `E2E-LOW-${stamp}-${i}`, 1))
    }
    const outId = await createStockedProduct(request, token, warehouse.id, `${prefix} out`, `E2E-OUT-${stamp}`, 0)
    // 500 pieces: never low by the shop's line, low by its own line of 1000.
    const ownLineId = await createStockedProduct(request, token, warehouse.id, `${prefix} own line`, `E2E-OWN-${stamp}`, 500)
    ids.push(outId, ownLineId)
    await api(request, token, `/pms/products/${ownLineId}/low-stock-threshold`, { method: 'PUT', body: { low_stock_threshold: 1000 } })

    await page.goto(`/inventory?stock=low&q=${encodeURIComponent(prefix)}`)
    await expect(page.getByText('23 products')).toBeVisible()
    await expect(page.getByRole('row').filter({ hasText: prefix })).toHaveCount(20)

    await page.getByRole('button', { name: /next/i }).click()
    await expect(page.getByRole('row').filter({ hasText: prefix })).toHaveCount(3)

    await page.getByRole('button', { name: 'Out of stock' }).click()
    await expect(page.getByText('1 product', { exact: true })).toBeVisible()
    await expect(page.getByRole('row').filter({ hasText: `${prefix} out` })).toBeVisible()
  } finally {
    for (const id of ids) await api(request, token, `/pms/products/${id}`, { method: 'DELETE' }).catch(() => undefined)
  }
})
