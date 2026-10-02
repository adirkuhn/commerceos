import { test, expect, api } from './fixtures'
import type { Page } from '@playwright/test'

interface Warehouse { id: string; name: string }
interface Product { id: string; has_variants: boolean }
interface Variant { id: string; name: string }
interface SKU { id: string; sku_code: string; portioned_from_sku_id?: string | null; price_minor_units: number }
interface ProductDetail { product: Product; variants: (Variant & { skus: SKU[] })[] }
interface Batch { id: string; lot_number: string; expiry_date: string | null; warehouse_id: string; quantity: number; is_general_ledger: boolean }
interface StockLevel { total: number }

async function addLine(page: Page, quantity: string, grams: string) {
  const dialog = page.getByRole('dialog', { name: 'Split into packages' })
  await dialog.getByLabel('Packages').fill(quantity)
  await dialog.getByLabel('Weight (g)').fill(grams)
  await dialog.getByLabel('Weight (g)').press('Enter')
}

// The Carne Seca flow: a simple product sold by the kilo, received as one
// lot, split from that lot's row into weighed packages. Each weight must
// become its own variant (one SKU per variant is what channel pushes
// need), carry the lot's expiry, and a later split of a known weight must
// restock it rather than fail on a duplicate SKU code.
test('split a weight-pool lot into packages: one variant per weight, restocked next time', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const name = `E2E Carne Seca ${stamp}`
  const lot = `E2E-${stamp}`
  const expiry = '2030-10-10'
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  try {
    const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name } })
    const bulk = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
      method: 'POST',
      body: {
        sku_code: `E2E-CS-${stamp}`, price_minor_units: 2999, currency_code: 'EUR', status: 'active', tracks_expiry: true,
        unit_type: 'kilo', pricing_mode: 'per_weight', price_per_kilo_minor_units: 29990, scale_code: `29${String(stamp).slice(-5)}`,
      },
    })
    const bulkLot = await api<Batch>(request, token, `/inventory/skus/${bulk.id}/batches`, {
      method: 'POST', body: { warehouse_id: warehouse.id, lot_number: lot, expiry_date: expiry },
    })
    await api(request, token, `/inventory/skus/${bulk.id}/movements`, {
      method: 'POST', body: { warehouse_id: warehouse.id, batch_id: bulkLot.id, type: 'receipt', quantity_delta: 5000 },
    })

    await page.goto(`/inventory?q=${encodeURIComponent(name)}`)
    await page.getByText(name).first().click()
    await page.getByRole('cell', { name: lot, exact: true }).click()
    await page.getByRole('button', { name: 'Split into packages' }).click()

    const dialog = page.getByRole('dialog', { name: 'Split into packages' })
    await expect(dialog.getByLabel('From lot')).toHaveValue(`${warehouse.id}|${bulkLot.id}`)
    await addLine(page, '2', '380')
    await addLine(page, '1', '430')
    await addLine(page, '1', '380')
    await expect(dialog.getByText('3 × 380g')).toBeVisible()
    await expect(dialog.getByText(/2 weights · 4 packages · 1\.57 kg of 5\.00 kg/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Create packages' }).click()

    const done = page.getByRole('dialog', { name: 'Packages created' })
    await expect(done.getByText('+3 · new')).toBeVisible()
    await expect(done.getByText('+1 · new')).toBeVisible()
    await done.getByRole('button', { name: 'Close' }).last().click()

    const detail = await api<ProductDetail>(request, token, `/pms/products/${product.id}`, { method: 'GET' })
    expect(detail.product.has_variants).toBe(true)
    const v380 = detail.variants.find(v => v.name === '~380g')
    expect(detail.variants.find(v => v.name === '~430g')).toBeTruthy()
    expect(v380?.skus).toHaveLength(1)
    const sku380 = v380!.skus[0]
    expect(sku380).toMatchObject({ sku_code: `E2E-CS-${stamp}-380G`, portioned_from_sku_id: bulk.id, price_minor_units: 11396 })
    expect((await api<StockLevel>(request, token, `/inventory/skus/${sku380.id}/stock-level`, { method: 'GET' })).total).toBe(3)
    const packageLot = (await api<Batch[]>(request, token, `/inventory/skus/${sku380.id}/batches`, { method: 'GET' })).find(b => !b.is_general_ledger)
    expect(packageLot).toMatchObject({ lot_number: lot, warehouse_id: warehouse.id, quantity: 3 })
    expect(packageLot?.expiry_date?.slice(0, 10)).toBe(expiry)
    const bulkAfter = (await api<Batch[]>(request, token, `/inventory/skus/${bulk.id}/batches`, { method: 'GET' })).find(b => b.id === bulkLot.id)
    expect(bulkAfter?.quantity).toBe(5000 - 3 * 380 - 430)

    // A later session: the same weight restocks its package, same lot batch.
    await page.goto(`/inventory?q=${encodeURIComponent(name)}`)
    await page.getByText(name).first().click()
    await page.getByRole('cell', { name: lot, exact: true }).first().click()
    await page.getByRole('button', { name: 'Split into packages' }).click()
    await addLine(page, '2', '380')
    await page.getByRole('dialog', { name: 'Split into packages' }).getByRole('button', { name: 'Create packages' }).click()
    await expect(page.getByRole('dialog', { name: 'Packages created' }).getByText('+2 · restocked')).toBeVisible()

    expect((await api<StockLevel>(request, token, `/inventory/skus/${sku380.id}/stock-level`, { method: 'GET' })).total).toBe(5)
    const packageLots = (await api<Batch[]>(request, token, `/inventory/skus/${sku380.id}/batches`, { method: 'GET' })).filter(b => !b.is_general_ledger)
    expect(packageLots).toHaveLength(1)
  } finally {
    await api(request, token, `/pms/products/${product.id}`, { method: 'DELETE' }).catch(() => undefined)
  }
})
