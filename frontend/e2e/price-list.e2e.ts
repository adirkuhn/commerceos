import { readFile } from 'node:fs/promises'
import { test, expect, api } from './fixtures'

interface Warehouse { id: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }

// Price list with a discount, then a B2B quote of the same product: typed
// quantity and price, below-cost highlight, totals and the CSV export.
test('price list: discounted prices and a B2B quote exported as CSV', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const name = `E2E Price list ${stamp}`
  const code = `E2E-PL-${stamp}`
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name, status: 'active' } })
  try {
    const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
    const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
      method: 'POST', body: { sku_code: code, price_minor_units: 1000, currency_code: 'EUR', status: 'active' },
    })
    await api(request, token, `/pms/skus/${sku.id}/cost-price`, { method: 'PUT', body: { cost_price_minor_units: 400 } })
    await api(request, token, `/inventory/skus/${sku.id}/movements`, {
      method: 'POST', body: { warehouse_id: warehouse.id, type: 'receipt', quantity_delta: 25 },
    })

    await page.goto(`/price-list?discount=30&q=${encodeURIComponent(name)}`)
    const row = page.getByTestId(`price-row-${sku.id}`)
    await expect(row).toContainText('€10.00')
    await expect(row).toContainText('€7.00')
    await expect(row).toContainText('€4.00')
    await expect(row).toContainText('25')

    await page.getByRole('tab', { name: 'B2B quote' }).click()
    await page.getByRole('textbox', { name: /search products/i }).fill(code)
    await page.getByRole('button', { name: new RegExp(`add .*${code}`, 'i') }).click()

    const line = page.getByTestId(`quote-line-${sku.id}`)
    await expect(line.getByRole('textbox', { name: `Unit price ${code}` })).toHaveValue('7.00')
    await line.getByRole('spinbutton', { name: `Qty ${code}` }).fill('30')
    await expect(page.getByTestId('quote-total')).toContainText('€210.00')
    await expect(line.getByRole('cell').nth(2)).toHaveAttribute('title', 'Only 25 available')

    // Below cost: always flagged, whatever the shop's markup target.
    const price = line.getByRole('textbox', { name: `Unit price ${code}` })
    await price.fill('3.50')
    await price.press('Enter')
    await expect(line).toHaveClass(/bg-danger/)
    await expect(page.getByTestId('quote-total')).toContainText('€105.00')

    // Survives a refresh: the quote lives in the URL.
    await page.reload()
    await expect(page.getByTestId('quote-total')).toContainText('€105.00')

    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: 'Export CSV' }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/^quote-\d{4}-\d{2}-\d{2}\.csv$/)
    const csv = (await readFile(await download.path())).toString('utf8').replace(/^﻿/, '')
    const [header, first] = csv.split('\r\n')
    expect(header.split(/[,;]/)).toEqual(['Product', 'Quantity', 'Unit price'])
    expect(first).toContain(code)
    expect(first).toMatch(/[,;]30[,;]3[.,]50$/)
  } finally {
    await api(request, token, `/pms/products/${product.id}`, { method: 'DELETE' }).catch(() => undefined)
  }
})
