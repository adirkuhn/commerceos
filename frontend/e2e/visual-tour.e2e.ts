import { devices } from '@playwright/test'
import { test, expect } from './fixtures'

// Screenshot tour for human review (no pixel assertions): the main screens
// in both appearances, plus the scanner at phone size. Images land in
// e2e-screenshots/. Each page must render without console errors.
const PAGES: [string, string][] = [
  ['dashboard', '/'],
  ['sales-orders', '/sales-orders'],
  ['customers', '/customers'],
  ['picking', '/picking'],
  ['products', '/products'],
  ['inventory', '/inventory'],
  ['suppliers', '/suppliers'],
  ['reports-overview', '/reports'],
  ['reports-sales-trends', '/reports/sales-trends'],
  ['settings', '/settings'],
]

for (const design of ['depot', 'classic'] as const) for (const appearance of ['light', 'dark'] as const) {
  test.describe(`${design} design, ${appearance} appearance`, () => {
    test.use({ appearance, design })

    for (const [name, path] of PAGES) {
      test(`${name}`, async ({ page }) => {
        const errors: string[] = []
        page.on('pageerror', e => errors.push(e.message))
        page.on('console', m => { if (m.type() === 'error') errors.push(m.text()) })
        // lm2 (team members) doesn't allow the dev-server origin -- an
        // environment limit, not a page error.
        const pageErrors = () => errors.filter(e => !e.includes('lm2.dev.localhost') && !e.includes('ERR_FAILED'))

        // Until the backend image with GET /dashboard/today is deployed the
        // live API 404s it -- show sample counts then so Today still renders.
        await page.route('**/dashboard/today', async route => {
          const live = await route.fetch()
          // This route replaces the fixture's, so it adds the CORS header too.
          const headers = { ...live.headers(), 'access-control-allow-origin': (await route.request().headerValue('origin')) ?? '*' }
          if (live.status() !== 404) return route.fulfill({ response: live, headers })
          return route.fulfill({
            response: live, status: 200, headers,
            json: { data: { to_pick: 14, to_pack: 6, stock_issues: 1, unmatched_skus: 2, expiring_batches: 4, expired_batches: 0 }, error: null },
          })
        })

        await page.setViewportSize({ width: 1440, height: 900 })
        await page.goto(path)
        await page.waitForLoadState('networkidle')
        await page.screenshot({ path: `e2e-screenshots/${design}-${appearance}-${name}.png` })
        expect(pageErrors(), pageErrors().join('\n')).toEqual([])
      })
    }

    test('command palette', async ({ page }) => {
      await page.setViewportSize({ width: 1440, height: 900 })
      await page.goto('/')
      await page.keyboard.press('ControlOrMeta+k')
      await page.getByRole('combobox', { name: 'Search' }).fill('pro')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: `e2e-screenshots/${design}-${appearance}-palette.png` })
    })
  })
}

test.describe('scanner on a phone', () => {
  const { viewport, deviceScaleFactor, isMobile, hasTouch, userAgent } = devices['iPhone 13']
  test.use({ viewport, deviceScaleFactor, isMobile, hasTouch, userAgent })

  test('launcher', async ({ page }) => {
    await page.goto('/scanner')
    await expect(page.getByText('CommerceOS Scanner')).toBeVisible()
    await page.screenshot({ path: 'e2e-screenshots/scanner-launcher.png' })
  })

  test('inventory search', async ({ page }) => {
    await page.goto('/scanner')
    await page.getByText('Inventory').click()
    await page.waitForLoadState('networkidle')
    await page.screenshot({ path: 'e2e-screenshots/scanner-inventory.png' })
  })
})
