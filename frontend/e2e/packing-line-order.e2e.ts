import { test, expect, api } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

interface Warehouse { id: string }
interface Product { id: string }
interface ProductDetail { variants: { skus?: { id: string; sku_code: string }[] }[] }
interface SalesOrderDetail { sales_order: { id: string } }
interface SKU { id: string; sku_code: string }

async function distinctSKUs(request: APIRequestContext, token: string, count: number): Promise<SKU[]> {
  const products = await api<Product[]>(request, token, '/pms/products?per_page=50', { method: 'GET' })
  const skus: SKU[] = []
  for (const p of products) {
    const detail = await api<ProductDetail>(request, token, `/pms/products/${p.id}`, { method: 'GET' })
    const sku = detail.variants.flatMap(v => v.skus ?? []).find(s => s.sku_code)
    if (sku) skus.push(sku)
    if (skus.length === count) break
  }
  return skus
}

// Packing keeps the work in reach: the order opens as it is, a partly
// packed line moves to the top, a finished line flashes and moves to the end.
test('packing: partly packed lines go on top, finished lines move to the end', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  const skus = await distinctSKUs(request, token, 3)
  test.skip(!warehouse || skus.length < 3, 'tenant needs a warehouse and three products with SKUs')

  const number = `E2E-ORDER-${Date.now()}`
  const names = ['E2E line A', 'E2E line B', 'E2E line C']
  const order = await api<SalesOrderDetail>(request, token, '/sales-orders', {
    method: 'POST',
    body: {
      number, status: 'pending', warehouse_id: warehouse.id, currency_code: 'EUR', occurred_at: new Date().toISOString(),
      line_items: [
        { sku_id: skus[0].id, name: names[0], quantity: 1, price_minor_units: 100 },
        { sku_id: skus[1].id, name: names[1], quantity: 2, price_minor_units: 100 },
        { sku_id: skus[2].id, name: names[2], quantity: 1, price_minor_units: 100 },
      ],
    },
  })
  const lineOrder = () => page.getByTestId('pack-line').allTextContents()
  const expectOrder = (expected: string[]) => expect.poll(async () => (await lineOrder()).map(text => names.find(n => text.includes(n)))).toEqual(expected)
  const scan = async (code: string) => {
    const input = page.getByPlaceholder('Scan barcode or type and press Enter...')
    await input.fill(code)
    await input.press('Enter')
  }
  try {
    await page.goto('/packing')
    await page.getByRole('button', { name: new RegExp(`#${number}`) }).click()
    await expectOrder(names)

    await scan(skus[1].sku_code)
    await expectOrder([names[1], names[0], names[2]])

    await scan(skus[0].sku_code)
    await expect(page.getByTestId('pack-line').filter({ hasText: names[0] })).toHaveAttribute('data-highlight', 'done')
    await expectOrder([names[1], names[2], names[0]])

    await page.reload()
    await expectOrder([names[1], names[2], names[0]])
  } finally {
    await api(request, token, `/sales-orders/${order.sales_order.id}`, { method: 'DELETE' }).catch(() => undefined)
  }
})
