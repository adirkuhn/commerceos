import { test, expect, api } from './fixtures'

interface Connection {
  id: string
  provider: string
  default_currency?: string | null
  default_warehouse_id?: string | null
  bol_economic_operator_id?: string | null
  bol_delivery_code?: string | null
  bol_shipping_cost_minor_units?: number
}
interface Product { id: string }
interface Variant { id: string }

// A valid EAN-13 (random first 12 digits + check digit), so the Bol tab
// can price it.
function randomEAN(): string {
  const digits = Array.from({ length: 12 }, () => Math.floor(Math.random() * 10))
  const sum = digits.reduce((s, d, i) => s + d * (i % 2 === 0 ? 1 : 3), 0)
  return digits.join('') + String((10 - (sum % 10)) % 10)
}

// The Bol connection gets a per-parcel shipping cost; a product's Bol tab
// shows the automatic price (ending .49/.99), bulk discount starts with
// 2/4/6 levels (Bol's max of 3), a level not cheaper than the one before is
// refused, and the levels survive a save + reload. The product stays a draft, so nothing is ever
// pushed to the real Bol; a typed fee % avoids Bol's live fee lookup.
test('bol pricing: shipping cost, automatic price and bulk discount levels', async ({ page, request, token }) => {
  const connections = await api<Connection[]>(request, token, '/integrations/connections', { method: 'GET' })
  const bol = connections.find(c => c.provider === 'bol')
  test.skip(!bol, 'tenant needs a Bol connection')
  if (!bol) return

  const settings = (shipping: number) => ({
    default_currency: bol.default_currency ?? '', default_warehouse_id: bol.default_warehouse_id ?? null,
    bol_economic_operator_id: bol.bol_economic_operator_id ?? null, bol_delivery_code: bol.bol_delivery_code ?? null,
    bol_shipping_cost_minor_units: shipping,
  })
  const stamp = Date.now()
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name: `E2E Bol pricing ${stamp}`, status: 'draft' } })
  try {
    const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
    await api(request, token, `/pms/variants/${variant.id}/skus`, {
      method: 'POST', body: { sku_code: `E2E-BOL-${stamp}`, barcode: randomEAN(), price_minor_units: 2500, currency_code: 'EUR', status: 'active' },
    })
    await api(request, token, `/integrations/connections/${bol.id}/settings`, { method: 'PATCH', body: settings(495) })

    await page.goto(`/products/${product.id}?tab=bol`)
    await expect(page.getByText(/Shipping per parcel 4\.95/)).toBeVisible()
    await page.getByLabel('Bol fee %').fill('15')

    const table = page.getByRole('table')
    const firstPrice = table.getByRole('row').nth(1).getByRole('cell').nth(1)
    await expect(firstPrice).toContainText(/\d+\.(49|99)/)

    await page.getByLabel('Bulk discount').check()
    await expect(page.getByLabel(/^From units/)).toHaveCount(3)
    await expect(table.getByRole('row')).toHaveCount(5)
    await expect(page.getByRole('button', { name: 'Add level' })).toHaveCount(0)

    const firstLevel = Number((await firstPrice.innerText()).match(/\d+\.\d{2}/)?.[0])
    await page.getByLabel('Price each 1').fill(firstLevel.toFixed(2))
    await expect(page.getByText(/lower than the price before/)).toBeVisible()

    await page.getByLabel('Price each 1').fill('')
    await page.getByLabel('From units 3').fill('10')
    await expect(table.getByRole('row').last()).toContainText('10+')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeEnabled()

    await page.reload()
    await expect(page.getByLabel('Bulk discount')).toBeChecked()
    await expect(page.getByLabel('From units 3')).toHaveValue('10')
    await expect(page.getByLabel('Bol fee %')).toHaveValue('15.00')
  } finally {
    await api(request, token, `/integrations/connections/${bol.id}/settings`, {
      method: 'PATCH', body: settings(bol.bol_shipping_cost_minor_units ?? 0),
    }).catch(() => undefined)
    await api(request, token, `/pms/products/${product.id}`, { method: 'DELETE' }).catch(() => undefined)
  }
})
