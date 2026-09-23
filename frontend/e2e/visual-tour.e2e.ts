import { devices } from '@playwright/test'
import { test, expect } from './fixtures'

// Screenshot tour for human review (no pixel assertions): the main screens
// in both appearances, plus the scanner at phone size. Images land in
// e2e-screenshots/. Each page must render without console errors.
const PAGES: [string, string][] = [
  ['dashboard', '/'],
  ['sales-orders', '/sales-orders'],
  ['products', '/products'],
  ['inventory', '/inventory'],
  ['suppliers', '/suppliers'],
  ['reports-overview', '/reports'],
  ['reports-sales-trends', '/reports/sales-trends'],
  ['settings', '/settings'],
]

for (const appearance of ['light', 'dark'] as const) {
  test.describe(`${appearance} appearance`, () => {
    test.use({ appearance })

    for (const [name, path] of PAGES) {
      test(`${name}`, async ({ page }) => {
        const errors: string[] = []
        page.on('pageerror', e => errors.push(e.message))
        page.on('console', m => { if (m.type() === 'error') errors.push(m.text()) })

        await page.setViewportSize({ width: 1440, height: 900 })
        await page.goto(path)
        await page.waitForLoadState('networkidle')
        await page.screenshot({ path: `e2e-screenshots/${appearance}-${name}.png` })
        expect(errors, errors.join('\n')).toEqual([])
      })
    }

    test('command palette', async ({ page }) => {
      await page.setViewportSize({ width: 1440, height: 900 })
      await page.goto('/')
      await page.keyboard.press('ControlOrMeta+k')
      await page.getByRole('combobox', { name: 'Search' }).fill('pro')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: `e2e-screenshots/${appearance}-palette.png` })
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
