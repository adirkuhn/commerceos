import { test, expect, api } from './fixtures'

interface Supplier { id: string }
interface Warehouse { id: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string; cost_price_minor_units?: number | null }
interface ProductDetail { variants: { skus: SKU[] }[] }
interface PurchaseOrder { id: string }
interface Booking { id: string }
interface SupplierCosts {
  last: { landed_unit_cost_minor_units: number } | null
  average_cost_minor_units: number | null
  by_supplier: { supplier_id: string | null; average_cost_minor_units: number }[]
}

// COM-11: two suppliers deliver the same product. Supplier A: 10 x 1.50
// plus 1.00 shipping (landed 1.60); then supplier B: 10 x 2.00. The cost
// price follows the last receipt (2.00), the average is weighted by
// quantity (1.80), each supplier keeps its own, and the product page shows
// it all with the history filterable per supplier.
test('supplier costs: cost price is the last receipt, averages per supplier and overall', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const supplierA = await api<Supplier>(request, token, '/suppliers', { method: 'POST', body: { name: `E2E Cost A ${stamp}`, status: 'active' } })
  const supplierB = await api<Supplier>(request, token, '/suppliers', { method: 'POST', body: { name: `E2E Cost B ${stamp}`, status: 'active' } })
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name: `E2E Cost Guarana ${stamp}`, status: 'active' } })
  const purchaseOrders: string[] = []
  try {
    const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
    const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
      method: 'POST', body: { sku_code: `E2E-SC-${stamp}`, price_minor_units: 399, currency_code: 'EUR', status: 'active' },
    })

    const receive = async (supplierId: string, unitCost: number, shipping: number) => {
      const po = await api<PurchaseOrder>(request, token, '/procurement/purchase-orders', {
        method: 'POST', body: { supplier_id: supplierId, warehouse_id: warehouse.id, currency_code: 'EUR' },
      })
      purchaseOrders.push(po.id)
      await api(request, token, `/procurement/purchase-orders/${po.id}/line-items`, {
        method: 'POST', body: { sku_id: sku.id, raw_description: `Guarana 2L ${stamp}`, quantity: 10, unit_cost_minor_units: unitCost },
      })
      if (shipping > 0) {
        await api(request, token, `/procurement/purchase-orders/${po.id}/line-items`, {
          method: 'POST', body: { raw_description: 'Shipping', quantity: 1, unit_cost_minor_units: shipping, line_type: 'shipping' },
        })
      }
      await api(request, token, `/procurement/purchase-orders/${po.id}`, {
        method: 'PATCH', body: { supplier_id: supplierId, warehouse_id: warehouse.id, status: 'sent', currency_code: 'EUR' },
      })
      const booking = await api<Booking>(request, token, `/procurement/purchase-orders/${po.id}/booking`, { method: 'POST', body: { warehouse_id: warehouse.id } })
      await api(request, token, `/procurement/bookings/${booking.id}/scan`, { method: 'POST', body: { sku_id: sku.id, quantity: 10 } })
      await api(request, token, `/procurement/bookings/${booking.id}/complete`, { method: 'POST' })
    }
    const costs = () => api<SupplierCosts>(request, token, `/pms/skus/${sku.id}/supplier-costs`, { method: 'GET' })

    await receive(supplierA.id, 150, 100)
    // the history is written from procurement.goods.received: wait for it
    await expect.poll(async () => (await costs()).last?.landed_unit_cost_minor_units, { timeout: 30_000 }).toBe(160)
    await receive(supplierB.id, 200, 0)
    await expect.poll(async () => (await costs()).last?.landed_unit_cost_minor_units, { timeout: 30_000 }).toBe(200)

    const summary = await costs()
    expect(summary.average_cost_minor_units).toBe(180)
    expect(Object.fromEntries(summary.by_supplier.map(s => [s.supplier_id, s.average_cost_minor_units])))
      .toEqual({ [supplierA.id]: 160, [supplierB.id]: 200 })
    const skuCostPrice = async () => {
      const detail = await api<ProductDetail>(request, token, `/pms/products/${product.id}`, { method: 'GET' })
      return detail.variants.flatMap(v => v.skus).find(s => s.id === sku.id)?.cost_price_minor_units
    }
    await expect.poll(skuCostPrice, { timeout: 30_000 }).toBe(200)

    await page.goto(`/products/${product.id}`)
    await expect(page.getByTestId('last-cost')).toContainText(/2[.,]00/)
    await expect(page.getByTestId('last-cost')).toContainText(`E2E Cost B ${stamp}`)
    await expect(page.getByTestId('average-cost')).toContainText(/1[.,]80/)
    await expect(page.getByTestId('supplier-row')).toHaveCount(2)
    await expect(page.getByTestId('history-row')).toHaveCount(2)

    await page.getByRole('button', { name: `E2E Cost A ${stamp}` }).click()
    await expect(page.getByTestId('history-row')).toHaveCount(1)
    await expect(page.getByTestId('history-row')).toContainText(/1[.,]60/)
    await page.getByRole('button', { name: /show all/ }).click()
    await expect(page.getByTestId('history-row')).toHaveCount(2)
  } finally {
    for (const id of purchaseOrders) await api(request, token, `/procurement/purchase-orders/${id}`, { method: 'DELETE' }).catch(() => {})
    await api(request, token, `/pms/products/${product.id}`, { method: 'DELETE' }).catch(() => {})
    await api(request, token, `/suppliers/${supplierA.id}`, { method: 'DELETE' }).catch(() => {})
    await api(request, token, `/suppliers/${supplierB.id}`, { method: 'DELETE' }).catch(() => {})
  }
})
