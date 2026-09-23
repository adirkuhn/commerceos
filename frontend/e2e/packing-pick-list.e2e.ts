import { test, expect, api } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

interface Warehouse { id: string }
interface Product { id: string }
interface ProductDetail { variants: { skus?: { id: string; sku_code: string }[] }[] }
interface SalesOrderDetail { sales_order: { id: string; number: string } }

async function firstSKU(request: APIRequestContext, token: string): Promise<{ id: string; sku_code: string } | null> {
  const products = await api<Product[]>(request, token, '/pms/products?per_page=20', { method: 'GET' })
  for (const p of products) {
    const detail = await api<ProductDetail>(request, token, `/pms/products/${p.id}`, { method: 'GET' })
    const sku = detail.variants.flatMap(v => v.skus ?? [])[0]
    if (sku) return sku
  }
  return null
}

// Picking is optional: an unpicked order shows in the packing queue, a
// multi-order pick list can be opened from it, and packing starts directly.
test.describe('packing without picking', () => {
  test('pick list for ticked orders, then pack an unpicked order', async ({ page, request, token }) => {
    const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
    const sku = await firstSKU(request, token)
    test.skip(!warehouse || !sku, 'tenant needs a warehouse and at least one SKU')

    const number = `E2E-PL-${Date.now()}`
    const order = await api<SalesOrderDetail>(request, token, '/sales-orders', {
      method: 'POST',
      body: {
        number, status: 'pending', warehouse_id: warehouse.id, currency_code: 'EUR', occurred_at: new Date().toISOString(),
        line_items: [{ sku_id: sku!.id, name: 'E2E pick list item', quantity: 2, price_minor_units: 100 }],
      },
    })
    try {
      await page.goto('/packing')
      const row = page.getByRole('button', { name: new RegExp(`#${number}`) })
      await expect(row).toBeVisible()
      await expect(row).toContainText('Not picked')

      await page.getByRole('checkbox', { name: `Add #${number} to pick list` }).check()
      await page.getByRole('button', { name: /Pick list \(1\)/ }).click()
      await expect(page.getByRole('heading', { name: 'Pick list' })).toBeVisible()
      await expect(page).toHaveURL(/[?&]pick=/)
      await expect(page.getByText(`#${number} × 2`)).toBeVisible()

      await page.reload()
      await expect(page.getByText(`#${number} × 2`)).toBeVisible()

      await page.goBack()
      await expect(page.getByRole('heading', { name: 'Pick list' })).toBeHidden()

      await page.getByRole('button', { name: new RegExp(`#${number}`) }).click()
      await expect(page).toHaveURL(/\/packing\/[^/?]+$/)
      await expect(page.getByText('0 / 2 items')).toBeVisible()
    } finally {
      await api(request, token, `/sales-orders/${order.sales_order.id}`, { method: 'DELETE' }).catch(() => undefined)
    }
  })
})
