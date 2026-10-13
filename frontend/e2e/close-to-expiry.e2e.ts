import { test, expect, api } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

interface Warehouse { id: string; name: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }
interface Batch { id: string }
interface CloseToExpiryList { windows: { window_days: number; min_days_left: number }[] }
interface Promotion { id: string; name: string }

function daysFromToday(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() + n)
  return d.toISOString().slice(0, 10)
}

async function createExpiringBatch(request: APIRequestContext, token: string, warehouseId: string, name: string, stamp: number, expiry: string) {
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
  const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
    method: 'POST', body: { sku_code: `E2E-CTE-${stamp}`, price_minor_units: 650, currency_code: 'EUR', status: 'active', tracks_expiry: true },
  })
  const batch = await api<Batch>(request, token, `/inventory/skus/${sku.id}/batches`, {
    method: 'POST', body: { warehouse_id: warehouseId, lot_number: `E2E-${stamp}`, expiry_date: expiry },
  })
  await api(request, token, `/inventory/skus/${sku.id}/movements`, {
    method: 'POST', body: { warehouse_id: warehouseId, batch_id: batch.id, type: 'receipt', quantity_delta: 5 },
  })
  return { productId: product.id, skuId: sku.id, batchId: batch.id }
}

// The close-to-expiry tab: a batch 20 days from expiry shows in its window,
// "Mark as seen" moves it to the Seen list, "Show again" brings it back,
// and "Put on sale" starts a clearance promotion the row then links to.
test('close-to-expiry: mark as seen, show again, put on sale', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')
  const { windows } = await api<CloseToExpiryList>(request, token, '/inventory/batches/close-to-expiry', { method: 'GET' })
  const window = windows.find(w => w.min_days_left <= 20 && w.window_days >= 20)
  test.skip(!window, 'no close-to-expiry window holds 20 days')

  const stamp = Date.now()
  const name = `E2E Close to expiry ${stamp}`
  const created = await createExpiringBatch(request, token, warehouse.id, name, stamp, daysFromToday(20))
  let promotionId = ''
  try {
    await page.goto(`/inventory/expiry?tab=expiring&window=${window!.window_days}`)
    const row = page.getByRole('listitem').filter({ hasText: name })
    await expect(row.getByRole('img', { name: /20 days left/i })).toBeVisible()

    await row.getByRole('button', { name: 'Mark as seen' }).click()
    await expect(row).toBeHidden()
    await page.getByRole('button', { name: /^Seen/ }).click()
    await expect(page).toHaveURL(/seen=1/)
    await row.getByRole('button', { name: 'Show again' }).click()
    await expect(row).toBeHidden()
    await page.getByRole('button', { name: /^To review/ }).click()

    // The lot has no cost of its own: the dialog judges the sale against
    // the SKU's average cost, and warns below it.
    await api(request, token, `/pms/skus/${created.skuId}/cost-price`, { method: 'PUT', body: { cost_price_minor_units: 250 } })
    await row.getByRole('button', { name: 'Put on sale' }).click()
    const dialog = page.getByRole('dialog', { name: `Put ${name} on sale` })
    const profit = dialog.getByRole('region', { name: 'Profit' })
    await expect(profit.getByText('average cost')).toBeVisible()
    await dialog.getByLabel('Sale price').fill('1')
    await expect(dialog.getByRole('alert')).toContainText('Below cost')
    await dialog.getByLabel('Sale price').fill('4,99')
    await expect(dialog.getByText(/Sells for €4\.99 instead of €6\.50/)).toBeVisible()
    await expect(profit.getByText(/Whole lot \(5 left\)/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Start sale' }).click()
    await expect(dialog).toBeHidden()
    await expect(page.getByRole('status')).toContainText('Sale started')
    const onSale = row.getByRole('link', { name: /On sale:/ })
    await expect(onSale).toBeVisible()
    promotionId = (await onSale.getAttribute('href'))?.split('/').pop() ?? ''

    const promotion = await api<Promotion>(request, token, `/promotions/${promotionId}`, { method: 'GET' })
    expect(promotion.name).toContain(name)
  } finally {
    if (promotionId) await api(request, token, `/promotions/${promotionId}`, { method: 'DELETE' }).catch(() => undefined)
    await api(request, token, `/pms/products/${created.productId}`, { method: 'DELETE' }).catch(() => undefined)
  }
})
