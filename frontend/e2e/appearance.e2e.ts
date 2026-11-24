import { test, expect } from './fixtures'

// Settings > Appearance: every change previews live on the whole app, a
// saved look survives a reload, and an unreadable accent can't be saved.
// Screenshots land in e2e-screenshots/.
for (const appearance of ['light', 'dark'] as const) {
  test.describe(`appearance settings, ${appearance}`, () => {
    test.use({ appearance, look: 'wallet' })

    test('restyles the admin for the whole shop', async ({ page }) => {
      // Until the backend image with /settings/appearance is deployed the
      // live API 404s it; then this test keeps the saved look itself.
      let stored: Record<string, unknown> | null = null
      await page.route('**/settings/appearance', async route => {
        const live = await route.fetch()
        const headers = { ...live.headers(), 'access-control-allow-origin': (await route.request().headerValue('origin')) ?? '*' }
        if (live.status() !== 404) return route.fulfill({ response: live, headers })
        if (route.request().method() === 'PUT') stored = { ...route.request().postDataJSON(), updated_at: new Date().toISOString() }
        const data = stored ?? { font: 'atkinson', accent_color: '#2F33B8', sidebar_color: '#14163F', corners: 'normal', text_size: 'normal', density: 'comfortable', updated_at: null }
        return route.fulfill({ response: live, status: 200, headers, json: { data, error: null } })
      })
      const cssVar = (name: string) => page.evaluate(n => getComputedStyle(document.documentElement).getPropertyValue(n).trim(), name)

      await page.setViewportSize({ width: 1440, height: 900 })
      await page.goto('/settings/appearance')
      await expect(page.getByRole('radio', { name: 'Atkinson Hyperlegible Next' })).toBeChecked()
      expect(await cssVar('--font-sans')).toContain('Atkinson Hyperlegible Next')
      await page.screenshot({ path: `e2e-screenshots/wallet-${appearance}-appearance-default.png` })

      await page.getByRole('radio', { name: 'IBM Plex Sans' }).check()
      await page.getByLabel('Accent colour hex').fill('#0B6E4F')
      await page.getByLabel('Sidebar colour hex').fill('#F4F1E8')
      await page.getByText('Round', { exact: true }).click()
      await page.getByText('Compact', { exact: true }).click()

      expect(await cssVar('--font-sans')).toContain('IBM Plex Sans')
      expect(await cssVar('--radius-lg')).toBe('10px')
      expect(await cssVar('--density')).toBe('0.8')
      expect(await cssVar('--rail')).toBe('244 241 232')
      await page.screenshot({ path: `e2e-screenshots/wallet-${appearance}-appearance-custom.png` })

      // Too light for white button text: blocked, with a darker shade offered.
      await page.getByLabel('Accent colour hex').fill('#FFB000')
      await expect(page.getByText('too light for white button text')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Save appearance' })).toBeDisabled()
      await page.screenshot({ path: `e2e-screenshots/wallet-${appearance}-appearance-contrast.png` })
      await page.getByRole('button', { name: /^Use #/ }).click()
      await expect(page.getByText('too light for white button text')).toBeHidden()

      await page.getByRole('button', { name: 'Save appearance' }).click()
      await expect(page.getByRole('status')).toHaveText('Saved')

      await page.reload()
      await expect(page.getByRole('radio', { name: 'IBM Plex Sans' })).toBeChecked()
      expect(await cssVar('--rail')).toBe('244 241 232')
      expect(await cssVar('--font-sans')).toContain('IBM Plex Sans')
      await page.goto('/sales-orders')
      await page.waitForLoadState('networkidle')
      await page.screenshot({ path: `e2e-screenshots/wallet-${appearance}-appearance-applied-sales-orders.png` })
    })
  })
}
