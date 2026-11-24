import { test, expect } from './fixtures'

// Inventory with a product opened: lot numbers as label chips, quantities as
// figures, best-before dates as stamps. Screenshots per look and appearance.
for (const look of ['wallet', 'graphite'] as const) for (const appearance of ['light', 'dark'] as const) {
  test.describe(`inventory lots, ${look} look, ${appearance} appearance`, () => {
    test.use({ look, appearance })

    test('opens a product and shows its lots', async ({ page }) => {
      await page.goto('/inventory')
      // Rows open on click; take the one with the most stock (it has lots).
      const row = page.locator('tbody tr').filter({ hasText: 'Carne Seca' }).first()
      await expect(row).toBeVisible()
      await row.click()
      await page.waitForTimeout(800)
      await page.screenshot({ path: `e2e-screenshots/${look}-${appearance}-inventory-lots.png` })
    })
  })
}
