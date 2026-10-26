import { test, expect, API_BASE_URL } from './fixtures'

// A company customer: the order's New customer dialog opens on Company, and
// the customer page shows client number, KVK and contact persons. The
// customer endpoints are served from fixtures so this runs before the
// backend with migrations 233/234 is deployed.
const API_ORIGIN = new URL(API_BASE_URL).origin
const json = (data: unknown) => ({ status: 200, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' }, body: JSON.stringify({ data, error: null }) })

const CLAUS = {
  id: 'cust-claus', tenant_id: 't', email: 'info@claus.nl', first_name: '', last_name: '', is_paying_customer: true,
  customer_type: 'company', display_name: 'Claus Hoofddorp B.V.', company_name: 'Claus Hoofddorp B.V.', client_number: '3286',
  kvk_number: '34073293', vat_number: 'NL001234567B01', date_created: '2026-10-01T10:00:00Z', date_modified: '2026-10-01T10:00:00Z',
  billing: { company: 'Claus Hoofddorp B.V.', address_1: 'Bosweg 19', postcode: '2131LX', city: 'Hoofddorp', country: 'NL' },
  shipping: { company: 'Claus Hoofddorp B.V.', address_1: 'Bosweg 19', postcode: '2131LX', city: 'Hoofddorp', country: 'NL' },
}
const KYLIAN = { id: 'cp-1', customer_id: 'cust-claus', first_name: 'Kylian', last_name: 'Klockenbrink', role: 'Employee', email: 'kylian@claus.nl', phone: '', created_at: '2026-10-01T10:00:00Z' }

test('the new customer dialog on an order opens on Private, switches to Company', async ({ page }) => {
  await page.goto('/sales-orders/new')
  await page.getByRole('button', { name: 'New customer' }).click()
  await expect(page.getByRole('radio', { name: 'Private' })).toHaveAttribute('aria-checked', 'true')
  await page.getByRole('radio', { name: 'Company' }).click()
  await page.getByLabel('Company name').fill('Claus Hoofddorp B.V.')
  await page.getByLabel('KVK number').fill('34073293')
  await page.getByLabel('Contact first name').fill('Kylian')
  await page.getByLabel('Contact last name').fill('Klockenbrink')
  await expect(page.getByRole('button', { name: 'Create customer' })).toBeEnabled()
  await page.screenshot({ path: 'e2e-screenshots/customer-company-dialog.png' })
})

test('a company customer page shows its details and contact persons', async ({ page }) => {
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/customers'), route => route.fulfill(json([CLAUS])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/customers/cust-claus/contact-persons'), route => route.fulfill(json([KYLIAN])))
  await page.route(url => url.origin === API_ORIGIN && url.pathname.endsWith('/customers/cust-claus/whatsapp-messages'), route => route.fulfill(json([])))

  await page.goto('/customers/cust-claus')
  await expect(page.getByRole('heading', { name: 'Claus Hoofddorp B.V.' })).toBeVisible()
  await expect(page.getByText('Client number: 3286')).toBeVisible()
  await expect(page.getByText('Kylian Klockenbrink')).toBeVisible()
  await page.screenshot({ path: 'e2e-screenshots/customer-company-detail.png', fullPage: true })
})
