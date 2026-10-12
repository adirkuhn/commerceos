import { test, expect, api } from './fixtures'

// The visual email editor: our own side panel (blocks / style / layers /
// variables) around the GrapesJS canvas, styled like the rest of the app.
// Screenshots per design and appearance land in e2e-screenshots/.
interface Template { id: string; name: string; channels: string[] }

for (const design of ['depot', 'classic'] as const) for (const appearance of ['light', 'dark'] as const) {
  test.describe(`email editor, ${design} design, ${appearance} appearance`, () => {
    test.use({ appearance, design })

    test('edits an email with the side panel', async ({ page, request, token }) => {
      const templates = await api<Template[]>(request, token, '/message-templates', { method: 'GET' })
      const template = templates.find(t => t.channels.includes('email'))
      test.skip(!template, 'no email template on this tenant')

      await page.setViewportSize({ width: 1440, height: 900 })
      await page.goto(`/message-center/templates/${template!.id}/design`)

      const blocks = page.getByTestId('panel-blocks')
      await expect(blocks.getByText('Store header')).toBeVisible({ timeout: 15_000 })
      await expect(blocks.getByText(/design\.blocks\./)).toHaveCount(0)
      // GrapesJS's own panel bar is gone.
      await expect(page.locator('.gjs-pn-panels')).toBeHidden()
      await page.screenshot({ path: `e2e-screenshots/${design}-${appearance}-email-editor-blocks.png` })

      // Selecting a block in the email opens its style settings.
      const frame = page.frameLocator('iframe.gjs-frame')
      await frame.getByText(/store_name/).first().click()
      await expect(page.getByRole('tab', { name: 'Style' })).toHaveAttribute('aria-selected', 'true')
      await expect(page.getByTestId('panel-styles').getByText('Text', { exact: true })).toBeVisible()
      await page.screenshot({ path: `e2e-screenshots/${design}-${appearance}-email-editor-style.png` })

      await page.getByRole('button', { name: 'Mobile' }).click()
      await expect(page.getByRole('button', { name: 'Mobile' })).toHaveAttribute('aria-pressed', 'true')
      await page.getByRole('tab', { name: 'Layers' }).click()
      await page.screenshot({ path: `e2e-screenshots/${design}-${appearance}-email-editor-mobile-layers.png` })
    })
  })
}
