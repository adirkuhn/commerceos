import { test, expect, API_BASE_URL } from './fixtures'

// The Integrations page scrolls to its last connection (it didn't: the
// shipping settings above the list pushed it off a page that couldn't
// scroll), and those shipping settings now live in Settings -> Shipping.
// Connections and shipping data are served from fixtures.
const API_ORIGIN = new URL(API_BASE_URL).origin
const json = (data: unknown) => ({ status: 200, headers: { 'content-type': 'application/json', 'access-control-allow-origin': '*' }, body: JSON.stringify({ data, error: null }) })
const isApi = (path: string) => (url: URL) => url.origin === API_ORIGIN && url.pathname.endsWith(path)

const PROVIDERS = ['woocommerce', 'bol', 'myparcel', 'postnl', 'whatsapp', 'stripe', 'revolut', 'email', 'moneybird', 'woocommerce', 'bol', 'myparcel']
const CONNECTIONS = PROVIDERS.map((provider, i) => ({
  id: `00000000-0000-4000-8000-0000000000${String(i).padStart(2, '0')}`, provider, store_url: `https://shop-${i}.example.com`,
  status: 'connected', created_at: '2026-10-01T00:00:00Z',
}))

test.beforeEach(async ({ page }) => {
  await page.route(isApi('/integrations/connections'), route => route.fulfill(json(CONNECTIONS)))
  await page.route(isApi('/shipping/connections'), route => route.fulfill(json({
    connections: [{ id: CONNECTIONS[2].id, provider: 'myparcel', status: 'active', created_at: '2026-10-01T00:00:00Z', is_default: true, missing: [] }],
    default_connection_id: CONNECTIONS[2].id, configured_connection_id: null,
  })))
  await page.route(isApi('/shipping/addresses'), route => route.fulfill(json({
    sender_address: null, return_address: null,
    company_address: { company: 'Cantinho Brasileiro', street: 'Purmerplein', number: '28A', postal_code: '1023BD', city: 'Amsterdam', cc: 'NL' },
  })))
})

test('the Integrations page scrolls to its last connection and links to the shipping settings', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 720 })
  await page.goto('/integrations')
  await expect(page.getByLabel('Default shipping provider')).toHaveCount(0)
  const lastCard = page.getByText(`https://shop-${CONNECTIONS.length - 1}.example.com`)
  await lastCard.scrollIntoViewIfNeeded()
  await expect(lastCard).toBeInViewport()
  await page.screenshot({ path: 'e2e-screenshots/integrations-scrolled.png' })

  await page.getByRole('link', { name: 'Open shipping settings' }).click()
  await expect(page).toHaveURL(/\/settings\/shipping$/)
  await expect(page.getByLabel('Default shipping provider')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Shipping addresses' })).toBeVisible()
  await page.screenshot({ path: 'e2e-screenshots/settings-shipping.png', fullPage: true })
})
