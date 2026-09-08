import { test, expect } from '@playwright/test'

// Exercises the real end-to-end multi-currency path: a shopper on a
// non-default-currency sales channel adds a product to their cart and
// checks out, and every price the backend returns along the way is in
// that channel's currency (see domain/fx + application/pms.ResolveSKUPrice
// -- this is the one thing no unit test can prove, since it needs a real
// SalesChannel.CurrencyCode row, a real fx.ExchangeRate (or override), and
// the actual HTTP contract, not a fake repo).
//
// Requires a seeded tenant -- there is no fixture-creation here, since
// admin (JWT/lm2) auth isn't scriptable from a storefront-scoped key.
// Before running, provision once via the admin UI/API:
//   1. An exchange rate for E2E_EXPECTED_CURRENCY (Settings > Localization).
//   2. An internal sales channel with that currency_code set at creation
//      (Sales Channels > Add sales channel) -- currency is immutable
//      after, so this can't be patched onto an existing channel.
//   3. At least one active product enrolled in it, in stock.
//   4. A storefront API key scoped to that channel (Settings > API keys).
// Then run with:
//   E2E_STOREFRONT_API_KEY=<key> E2E_EXPECTED_CURRENCY=USD npx playwright test
const STOREFRONT_API_KEY = process.env.E2E_STOREFRONT_API_KEY ?? ''
const EXPECTED_CURRENCY = process.env.E2E_EXPECTED_CURRENCY ?? 'USD'

test.describe('multi-currency checkout', () => {
  test.skip(!STOREFRONT_API_KEY, 'requires E2E_STOREFRONT_API_KEY for a seeded non-default-currency channel -- see this file\'s own doc comment')

  test('adding to cart and checking out both price in the channel\'s currency', async ({ page }) => {
    await page.goto('/products')

    const firstProduct = page.locator('a[href^="/products/"]').first()
    await expect(firstProduct).toBeVisible()
    await firstProduct.click()

    const addToCartResponse = page.waitForResponse(res => res.url().includes('/carts/') && res.request().method() === 'PUT')
    await page.getByRole('button', { name: 'Add to cart' }).first().click()
    const cartBody = await (await addToCartResponse).json()
    expect(cartBody.data.items[0].currency_code).toBe(EXPECTED_CURRENCY)

    await page.goto('/checkout')
    await page.getByLabel('Email').fill('e2e-shopper@example.com')
    await page.getByLabel('First name').fill('E2E')
    await page.getByLabel('Last name').fill('Shopper')
    await page.getByLabel('Address').fill('Main St 1')
    await page.getByLabel('City').fill('Amsterdam')
    await page.getByLabel('Postcode').fill('1000AA')
    await page.getByLabel('Country').fill('NL')

    const checkoutResponse = page.waitForResponse(res => res.url().includes('/checkout') && res.request().method() === 'POST');
    await page.getByRole('button', { name: 'Place order' }).click()
    const checkoutBody = await (await checkoutResponse).json()

    expect(checkoutBody.data.currency_code).toBe(EXPECTED_CURRENCY)
    expect(checkoutBody.data.total_minor_units).toBeGreaterThan(0)
    await expect(page).toHaveURL(/\/order-confirmation\//)
  })
})
