import { test, expect } from '@playwright/test'

// A real browser walking the full guest checkout flow against the mock
// repository layer -- the closest thing this app has to a production
// smoke test, since there is no real backend to hit yet.
test('guest can add a product to cart and complete checkout', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Gear for real days outside/ })).toBeVisible()

  await page.goto('/product/summit-shell-jacket')
  await expect(page.getByRole('heading', { name: 'Summit Shell Jacket' })).toBeVisible()

  await page.getByRole('radio', { name: 'M', exact: true }).click()
  await page.getByRole('radio', { name: 'Charcoal' }).click()
  await page.getByRole('button', { name: 'Add to cart' }).click()

  // Adding an item opens the cart drawer.
  await expect(page.getByRole('heading', { name: 'Your cart' })).toBeVisible()
  await page.getByRole('link', { name: 'Checkout' }).click()

  await expect(page).toHaveURL(/\/checkout$/)
  // React Router v7 wraps this client-side navigation in a transition, so
  // the previous route's content can stay on screen (still fully
  // interactive) until the checkout route's lazy chunk finishes loading --
  // waiting for a marker unique to CheckoutPage avoids racing that swap.
  await expect(page.locator('#checkout-email')).toBeVisible()
  await page.getByLabel('Email', { exact: true }).fill('e2e-guest@example.com')
  await page.getByRole('button', { name: 'Continue to shipping' }).click()

  await page.getByLabel('First name').fill('Morgan')
  await page.getByLabel('Last name').fill('Lee')
  await page.getByLabel('Street address').fill('482 Ridgeview Ave')
  await page.getByLabel('City').fill('Boulder')
  await page.getByLabel('Postal code').fill('80302')
  await page.getByRole('button', { name: 'Continue', exact: true }).click()
  await page.getByRole('button', { name: 'Continue to payment' }).click()

  await page.getByRole('button', { name: 'Review order' }).click()

  await page.getByLabel(/I agree to the/).check()
  await page.getByRole('button', { name: 'Place order' }).click()

  await expect(page).toHaveURL(/\/order-confirmation\//)
  await expect(page.getByRole('heading', { name: 'Thank you for your order!' })).toBeVisible()
})
