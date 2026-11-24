import { test, expect, api } from './fixtures'

// A customer's shipping address book against the real backend (migration
// 253): no hand-set "Paying customer" box, addresses added and made the
// default on the edit page, "Same as billing", and the order form's
// "Ship to" picker.
interface Customer { id: string; is_paying_customer: boolean; ship_to_billing: boolean; shipping?: { address_1?: string; city?: string } }
interface ShippingAddress { id: string; label: string; address_1: string; is_default: boolean }

const run = Date.now().toString(36)
const name = `E2E Ship ${run}`

test('a customer keeps several shipping addresses and can ship to billing', async ({ page, request, token }) => {
  const customer = await api<Customer>(request, token, '/customers', {
    method: 'POST',
    body: {
      first_name: 'E2E', last_name: `Ship ${run}`,
      billing: { address_1: 'Billingstraat 1', postcode: '1011AA', city: 'Amsterdam', country: 'NL' },
      shipping: { address_1: 'Thuisweg 2', postcode: '3511AB', city: 'Utrecht', country: 'NL' },
    },
  })
  try {
    expect(customer.is_paying_customer, 'no orders: not paying').toBe(false)

    await page.goto(`/customers/${customer.id}?edit=1`)
    await expect(page.getByRole('heading', { name })).toBeVisible()
    await expect(page.getByRole('checkbox', { name: 'Paying customer' })).toHaveCount(0)

    // The shipping address given at creation is the default one.
    const book = page.getByRole('heading', { name: 'Shipping addresses' }).locator('..')
    await expect(book.getByText('Thuisweg 2, 3511AB Utrecht, NL')).toBeVisible()
    await expect(book.getByText('Default', { exact: true })).toBeVisible()

    // Add a second one and make it the default.
    await book.getByRole('button', { name: 'Add shipping address' }).click()
    await book.getByLabel('Label').fill('Warehouse')
    await book.getByLabel('Address line 1').fill('Dokweg 9')
    await book.getByLabel('City').fill('Rotterdam')
    await book.getByRole('button', { name: 'Save' }).click()
    await expect(book.getByText('Warehouse')).toBeVisible()
    await book.getByRole('button', { name: 'Make Warehouse the default' }).click()
    await expect.poll(async () => {
      const list = await api<ShippingAddress[]>(request, token, `/customers/${customer.id}/shipping-addresses`, { method: 'GET' })
      return list.find(a => a.is_default)?.label
    }).toBe('Warehouse')
    expect((await api<Customer>(request, token, `/customers/${customer.id}`, { method: 'GET' })).shipping?.address_1,
      "the customer's shipping is the default address").toBe('Dokweg 9')
    await page.screenshot({ path: 'e2e-screenshots/customer-shipping-book.png', fullPage: true })

    // The order form offers both addresses, the default preselected.
    await page.goto('/sales-orders/new')
    await page.getByLabel('Customer').fill(`Ship ${run}`)
    await page.getByRole('button', { name: new RegExp(name) }).click()
    const shipTo = page.getByRole('combobox', { name: 'Ship to' })
    await expect(shipTo).toBeVisible()
    await expect(shipTo.locator('option:checked')).toHaveText(/Warehouse/)
    await shipTo.selectOption({ label: 'Thuisweg 2, 3511AB Utrecht, NL' })
    await expect(page.locator('#shipping-address_1')).toHaveValue('Thuisweg 2')
    await page.screenshot({ path: 'e2e-screenshots/order-ship-to.png', fullPage: true })

    // Same as billing: goods go to the billing address.
    await page.goto(`/customers/${customer.id}?edit=1`)
    await page.getByRole('checkbox', { name: 'Same as billing' }).check()
    await expect(page.getByText('Goods go to the billing address.')).toBeVisible()
    await page.getByRole('button', { name: 'Save', exact: true }).first().click()
    await expect.poll(async () => (await api<Customer>(request, token, `/customers/${customer.id}`, { method: 'GET' })).ship_to_billing).toBe(true)
    const saved = await api<Customer>(request, token, `/customers/${customer.id}`, { method: 'GET' })
    expect(saved.shipping?.address_1).toBe('Billingstraat 1')
    const list = await api<ShippingAddress[]>(request, token, `/customers/${customer.id}/shipping-addresses`, { method: 'GET' })
    expect(list.map(a => a.address_1).sort(), 'the book is kept').toEqual(['Dokweg 9', 'Thuisweg 2'])
  } finally {
    await api(request, token, `/customers/${customer.id}`, { method: 'DELETE' })
  }
})
