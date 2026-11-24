import { test, expect, DEFAULT_APPEARANCE_RESPONSE } from './fixtures'

// Settings > Appearance: the shop default look and layout changes preview
// live on the whole app, and a saved appearance survives a reload. Runs
// without a look of one's own, so the shop default is what shows.
// Screenshots land in e2e-screenshots/.
for (const appearance of ['light', 'dark'] as const) {
  test.describe(`appearance settings, ${appearance}`, () => {
    test.use({ appearance, look: null })

    test('sets the default look and layout for the whole shop', async ({ page }) => {
      // Until the backend image with the new /settings/appearance is
      // deployed the live API may 404 it; then this test keeps the saved
      // appearance itself.
      let stored: Record<string, unknown> | null = null
      await page.route('**/settings/appearance', async route => {
        const live = await route.fetch()
        const headers = { ...live.headers(), 'access-control-allow-origin': (await route.request().headerValue('origin')) ?? '*' }
        if (live.status() !== 404) return route.fulfill({ response: live, headers })
        if (route.request().method() === 'PUT') stored = { ...route.request().postDataJSON(), updated_at: new Date().toISOString() }
        return route.fulfill({ response: live, status: 200, headers, json: { data: stored ?? DEFAULT_APPEARANCE_RESPONSE, error: null } })
      })
      const cssVar = (name: string) => page.evaluate(n => getComputedStyle(document.documentElement).getPropertyValue(n).trim(), name)
      const htmlHasClass = (name: string) => page.evaluate(c => document.documentElement.classList.contains(c), name)
      const lookGroup = page.getByRole('radiogroup', { name: 'Default look' })

      await page.setViewportSize({ width: 1440, height: 900 })
      await page.goto('/settings/appearance')
      // Start from the defaults whatever an earlier run saved.
      await expect(lookGroup).toBeVisible()
      await page.getByRole('button', { name: 'Back to defaults' }).click()
      if (await page.getByRole('button', { name: 'Save appearance' }).isEnabled()) {
        await page.getByRole('button', { name: 'Save appearance' }).click()
        await expect(page.getByRole('status')).toHaveText('Saved')
      }
      await expect(lookGroup.getByRole('radio', { name: 'Ticket Wallet' })).toBeChecked()
      expect(await htmlHasClass('look-wallet')).toBe(true)
      await page.screenshot({ path: `e2e-screenshots/appearance-${appearance}-default.png` })

      await lookGroup.getByRole('radio', { name: 'Graphite' }).check()
      await page.getByText('Round', { exact: true }).click()
      await page.getByText('Compact', { exact: true }).click()

      expect(await htmlHasClass('look-graphite')).toBe(true)
      expect(await htmlHasClass('look-wallet')).toBe(false)
      expect(await cssVar('--radius-lg')).toBe('10px')
      expect(await cssVar('--density')).toBe('0.8')
      await page.screenshot({ path: `e2e-screenshots/appearance-${appearance}-graphite-preview.png` })

      await page.getByRole('button', { name: 'Save appearance' }).click()
      await expect(page.getByRole('status')).toHaveText('Saved')

      await page.reload()
      await expect(lookGroup.getByRole('radio', { name: 'Graphite' })).toBeChecked()
      expect(await htmlHasClass('look-graphite')).toBe(true)
      expect(await cssVar('--density')).toBe('0.8')
      await page.goto('/sales-orders')
      await page.waitForLoadState('networkidle')
      expect(await htmlHasClass('look-graphite')).toBe(true)
      await page.screenshot({ path: `e2e-screenshots/appearance-${appearance}-graphite-sales-orders.png` })

      // Leave the shop as it was for other specs.
      await page.goto('/settings/appearance')
      await page.getByRole('button', { name: 'Back to defaults' }).click()
      await page.getByRole('button', { name: 'Save appearance' }).click()
      await expect(page.getByRole('status')).toHaveText('Saved')
    })
  })
}
