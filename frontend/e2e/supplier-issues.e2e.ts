import { test, expect, api } from './fixtures'

interface Supplier { id: string }
interface Warehouse { id: string }
interface Product { id: string }
interface Variant { id: string }
interface SKU { id: string }
interface PurchaseOrder { id: string; currency_code: string }
interface Booking { id: string }
interface SupplierIssueList { issues: { kind: string; status: string }[] }

// A PO for 10 units: the delivery brings 6, 2 of which arrive damaged.
// The PO page lists 4 missing and 2 damaged; a note is saved, and emailing
// the supplier (prefilled from the list) marks both claimed.
test('supplier issues: list a delivery’s missing and damaged items, note one, email the supplier', async ({ page, request, token }) => {
  const [warehouse] = await api<Warehouse[]>(request, token, '/inventory/warehouses', { method: 'GET' })
  test.skip(!warehouse, 'tenant needs a warehouse')

  const stamp = Date.now()
  const supplier = await api<Supplier>(request, token, '/suppliers', {
    method: 'POST', body: { name: `E2E Supplier ${stamp}`, status: 'active', email: 'claims@example.com' },
  })
  const product = await api<Product>(request, token, '/pms/products', { method: 'POST', body: { name: `E2E Guarana ${stamp}`, status: 'active' } })
  let purchaseOrderId = ''
  try {
    const variant = await api<Variant>(request, token, `/pms/products/${product.id}/variants`, { method: 'POST', body: { name: 'Default' } })
    const sku = await api<SKU>(request, token, `/pms/variants/${variant.id}/skus`, {
      method: 'POST', body: { sku_code: `E2E-SI-${stamp}`, price_minor_units: 399, currency_code: 'EUR', status: 'active' },
    })
    const po = await api<PurchaseOrder>(request, token, '/procurement/purchase-orders', {
      method: 'POST', body: { supplier_id: supplier.id, warehouse_id: warehouse.id, currency_code: 'EUR' },
    })
    purchaseOrderId = po.id
    await api(request, token, `/procurement/purchase-orders/${po.id}/line-items`, {
      method: 'POST', body: { sku_id: sku.id, raw_sku: 'SUP-77', raw_description: `Guarana 2L ${stamp}`, quantity: 10, unit_cost_minor_units: 150 },
    })
    await api(request, token, `/procurement/purchase-orders/${po.id}`, {
      method: 'PATCH', body: { supplier_id: supplier.id, warehouse_id: warehouse.id, status: 'sent', currency_code: 'EUR' },
    })
    const booking = await api<Booking>(request, token, `/procurement/purchase-orders/${po.id}/booking`, { method: 'POST', body: { warehouse_id: warehouse.id } })
    await api(request, token, `/procurement/bookings/${booking.id}/scan`, { method: 'POST', body: { sku_id: sku.id, quantity: 4 } })
    await api(request, token, `/procurement/bookings/${booking.id}/scan`, {
      method: 'POST', body: { sku_id: sku.id, action_type: 'damaged', quantity: 2, reason: 'arrived crushed' },
    })
    await api(request, token, `/procurement/bookings/${booking.id}/complete`, { method: 'POST' })

    await page.goto(`/procurement/${po.id}`)
    const section = page.locator('div').filter({ has: page.getByRole('heading', { name: 'Supplier issues' }) }).last()
    const missing = section.getByRole('row', { name: `Guarana 2L ${stamp}` }).filter({ hasText: 'Missing' })
    const damaged = section.getByRole('row', { name: `Guarana 2L ${stamp}` }).filter({ hasText: 'Damaged' })
    await expect(missing).toContainText('ordered 10, received 6')
    await expect(missing).toContainText('4')
    await expect(damaged).toContainText('arrived crushed')
    // The section's hidden table header must not stretch the page with empty space.
    expect(await page.evaluate(() => document.documentElement.scrollHeight <= window.innerHeight)).toBe(true)

    await damaged.getByRole('button', { name: 'Add note' }).click()
    await damaged.getByRole('textbox').fill('photo sent')
    await damaged.getByRole('textbox').press('Enter')
    await expect(damaged.getByRole('button', { name: 'Edit note' })).toContainText('photo sent')

    await section.getByRole('button', { name: 'Email supplier' }).click()
    const dialog = page.getByRole('dialog', { name: 'Email supplier' })
    await expect(dialog.getByLabel('To')).toHaveValue('claims@example.com')
    await expect(dialog.getByLabel('Message')).toHaveValue(new RegExp(`- 4 × Guarana 2L ${stamp} \\(your code SUP-77\\) — ordered 10, received 6`))
    await expect(dialog.getByLabel('Message')).toHaveValue(/- 2 × .* — arrived crushed/)
    await dialog.getByRole('button', { name: 'Send' }).click()
    await expect(dialog).toBeHidden()
    await expect(section.getByRole('status')).toContainText('2 items marked as claimed')
    await expect(missing.getByRole('combobox')).toHaveValue('claimed')

    const list = await api<SupplierIssueList>(request, token, `/procurement/purchase-orders/${po.id}/supplier-issues`, { method: 'GET' })
    expect(list.issues.map(i => `${i.kind}:${i.status}`).sort()).toEqual(['damaged:claimed', 'missing:claimed'])
  } finally {
    if (purchaseOrderId) await api(request, token, `/procurement/purchase-orders/${purchaseOrderId}`, { method: 'DELETE' }).catch(() => {})
    await api(request, token, `/pms/products/${product.id}`, { method: 'DELETE' }).catch(() => {})
    await api(request, token, `/suppliers/${supplier.id}`, { method: 'DELETE' }).catch(() => {})
  }
})
