import { expect, request, test, type APIRequestContext } from '@playwright/test'
import { API_BASE_URL, CONFIGURED, NOT_CONFIGURED_REASON, allowCrossOriginAPI, ownerToken } from './fixtures'

// A shopper on a freshly published default theme: find a product, buy it,
// see the order; sign up for the newsletter. Also the search-engine side
// (structured data, sitemap). Seeds its own channel, theme and product.

test.skip(!CONFIGURED, NOT_CONFIGURED_REASON)
test.describe.configure({ mode: 'serial' })

let api: APIRequestContext
let channelId = ''
let productId = ''
let productSlug = ''
const stamp = Date.now()
const productName = `E2E Sneaker ${stamp}`

async function data<T>(res: Awaited<ReturnType<APIRequestContext['post']>>): Promise<T> {
  expect(res.ok(), await res.text()).toBeTruthy()
  return (await res.json()).data as T
}

test.beforeAll(async () => {
  api = await request.newContext({
    baseURL: `${API_BASE_URL}/`, ignoreHTTPSErrors: true,
    extraHTTPHeaders: { Authorization: `Bearer ${ownerToken()}` },
  })
  const [warehouse] = await data<{ id: string }[]>(await api.get('inventory/warehouses'))
  test.skip(!warehouse, 'tenant needs a warehouse')

  channelId = (await data<{ id: string }>(await api.post('integrations/sales-channels', {
    data: { name: `E2E shop ${stamp}`, type: 'internal', enrollment_mode: 'all', storefront_mode: 'theme', default_warehouse_id: warehouse.id },
  }))).id
  const theme = await data<{ id: string }>(await api.post('theme/themes', { data: { sales_channel_id: channelId, name: 'E2E default' } }))
  const detail = await data<{ draft: { revision: number } }>(await api.get(`theme/themes/${theme.id}`))
  await data(await api.post(`theme/themes/${theme.id}/publish`, { data: { expected_revision: detail.draft.revision } }))

  const product = await data<{ id: string; slug: string }>(await api.post('pms/products', {
    data: { name: productName, status: 'active', description: '<p>Light &amp; fast</p>' },
  }))
  productId = product.id
  productSlug = product.slug
  const variant = await data<{ id: string }>(await api.post(`pms/products/${product.id}/variants`, { data: { name: 'Default' } }))
  const sku = await data<{ id: string }>(await api.post(`pms/variants/${variant.id}/skus`, {
    data: { sku_code: `E2E-SNK-${stamp}`, price_minor_units: 1250, currency_code: 'EUR', status: 'active' },
  }))
  await data(await api.post(`inventory/skus/${sku.id}/movements`, {
    data: { warehouse_id: warehouse.id, type: 'receipt', quantity_delta: 5 },
  }))
})

test.afterAll(async () => {
  if (productId) await api.delete(`pms/products/${productId}`).catch(() => undefined)
  if (channelId) await api.delete(`integrations/sales-channels/${channelId}`).catch(() => undefined)
  await api?.dispose()
})

test('a shopper finds a product, checks out and sees the order', async ({ page }) => {
  await allowCrossOriginAPI(page)
  await page.goto(`/store/${channelId}/search?q=${encodeURIComponent(productName)}`)
  await page.getByRole('link', { name: productName }).first().click()
  await expect(page).toHaveURL(new RegExp(`/products/${productSlug}$`))
  await expect(page.getByRole('heading', { name: productName })).toBeVisible()

  await page.getByRole('button', { name: 'Add to cart' }).click()
  await page.goto(`/store/${channelId}/cart`)
  await expect(page.locator('.cos-cart-items').getByText(productName)).toBeVisible()
  await page.locator('main').getByRole('link', { name: 'Checkout' }).click()

  const form = page.locator('form.cos-checkout')
  await form.getByLabel('Email').fill(`e2e+${stamp}@example.com`)
  await form.getByLabel('First name').fill('Ada')
  await form.getByLabel('Last name').fill('Lovelace')
  await form.getByLabel('Street and number').fill('Damrak 1')
  await form.getByLabel('Postcode').fill('1012 LG')
  await form.getByLabel('City').fill('Amsterdam')
  await form.getByRole('button', { name: 'Place order' }).click()

  // No payment provider on this channel: straight to the order page,
  // which shows the order to the shopper who just placed it.
  await expect(page).toHaveURL(/\/orders\/[^/]+$/)
  await expect(page.getByRole('heading', { name: 'Thank you for your order' })).toBeVisible()
  await expect(page.locator('.cos-order-status').getByText('Total')).toBeVisible()
})

test('a shopper signs up for the newsletter and is asked to confirm by email', async ({ page }) => {
  await allowCrossOriginAPI(page)
  await page.goto(`/store/${channelId}/`)
  const signup = page.locator('.cos-newsletter')
  await signup.getByLabel('Email').fill(`news+${stamp}@example.com`)
  await signup.getByRole('checkbox').check()
  await signup.getByRole('button', { name: 'Subscribe' }).click()
  await expect(signup.getByRole('status')).toContainText('confirm your subscription')
})

test('a stale newsletter confirmation link explains itself and stays out of search engines', async ({ page }) => {
  await allowCrossOriginAPI(page)
  const res = await page.goto(`/store/${channelId}/newsletter/confirm?token=not-a-real-token`)
  expect(res?.status()).toBe(200)
  await expect(page.getByRole('alert')).toContainText('expired or was already used')
  await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex')
})

test('search engines get structured data and a sitemap', async ({ page, request: http }) => {
  await page.goto(`/store/${channelId}/products/${productSlug}`)
  const ld = await page.locator('script[type="application/ld+json"]').allTextContents()
  const product = ld.map(t => JSON.parse(t)).find(d => d['@type'] === 'Product')
  expect(product).toMatchObject({ name: productName, description: 'Light & fast', offers: { price: '12.50', priceCurrency: 'EUR' } })
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', new RegExp(`/store/${channelId}/products/${productSlug}$`))

  const sitemap = await http.get(`/store/${channelId}/sitemap.xml`)
  expect(sitemap.status()).toBe(200)
  expect(await sitemap.text()).toContain(`/store/${channelId}/products/${productSlug}</loc>`)
})
