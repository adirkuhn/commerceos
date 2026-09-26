import { test, expect, api } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

interface Warehouse { id: string; name: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }
interface Batch { id: string; lot_number: string; expiry_date: string | null; warehouse_id: string; quantity: number; is_general_ledger: boolean }

async function createProductWithSKU(
  request: APIRequestContext, token: string, name: string, sku: Record<string, unknown>,
): Promise<{ productId: string; skuId: string }> {
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
  const created = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
    method: 'POST', body: { price_minor_units: 100, currency_code: 'EUR', status: 'active', ...sku },
  })
  return { productId: product.id, skuId: created.id }
}

// Splitting a case from a batch row: the units must land in a batch with
// the same lot and expiry on the unit SKU, and the unit product's row
// total must update on screen without a reload.
test('split one case from a batch row: units keep the lot and the unit row refreshes', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const prefix = `E2E Split ${stamp}`
  const lot = `E2E-${stamp}`
  const expiry = '2027-03-30'
  const unit = await createProductWithSKU(request, token, `${prefix} unit`, { sku_code: `E2E-UNIT-${stamp}`, tracks_expiry: true })
  const caseProduct = await createProductWithSKU(request, token, `${prefix} case`, {
    sku_code: `E2E-CASE-${stamp}`, tracks_expiry: true, case_of_sku_id: unit.skuId, units_per_case: 12,
  })
  try {
    const caseBatch = await api<Batch>(request, token, `/inventory/skus/${caseProduct.skuId}/batches`, {
      method: 'POST', body: { warehouse_id: warehouse.id, lot_number: lot, expiry_date: expiry },
    })
    await api(request, token, `/inventory/skus/${caseProduct.skuId}/movements`, {
      method: 'POST', body: { warehouse_id: warehouse.id, batch_id: caseBatch.id, type: 'receipt', quantity_delta: 3 },
    })

    await page.goto(`/inventory?q=${encodeURIComponent(prefix)}`)
    const unitRow = page.getByRole('row').filter({ hasText: `${prefix} unit` })
    await expect(unitRow.getByRole('cell').nth(5)).toHaveText('0')

    await page.getByText(`${prefix} case`).click()
    await page.getByRole('cell', { name: lot, exact: true }).click()
    await page.getByRole('button', { name: 'Split case into units' }).click()

    const dialog = page.getByRole('dialog', { name: 'Split case into units' })
    await expect(dialog.getByText(warehouse.name)).toBeVisible()
    await expect(dialog.getByText(lot)).toBeVisible()
    await dialog.getByLabel('Cases to split').fill('1')
    await dialog.getByRole('button', { name: 'Split', exact: true }).click()
    await expect(dialog).toBeHidden()

    // No reload: the other product's row picks the new units up by itself.
    await expect(unitRow.getByRole('cell').nth(5)).toHaveText('12')

    const unitBatches = await api<Batch[]>(request, token, `/inventory/skus/${unit.skuId}/batches`, { method: 'GET' })
    const unitBatch = unitBatches.find(b => !b.is_general_ledger)
    expect(unitBatch).toMatchObject({ lot_number: lot, warehouse_id: warehouse.id, quantity: 12 })
    expect(unitBatch?.expiry_date?.slice(0, 10)).toBe(expiry)
  } finally {
    for (const id of [caseProduct.productId, unit.productId]) {
      await api(request, token, `/pms/products/${id}`, { method: 'DELETE' }).catch(() => undefined)
    }
  }
})
