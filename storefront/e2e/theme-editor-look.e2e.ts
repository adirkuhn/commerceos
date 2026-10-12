import { expect, request, test, type APIRequestContext, type Page } from '@playwright/test'
import { API_BASE_URL, CONFIGURED, NOT_CONFIGURED_REASON, allowCrossOriginAPI, ownerToken } from './fixtures'

// The theme editor's own look: Add / Layers panel, a readable selected
// layer, the admin's light/dark choice. Screenshots for review land in
// e2e-screenshots/.
test.skip(!CONFIGURED, NOT_CONFIGURED_REASON)
test.describe.configure({ mode: 'serial' })

let api: APIRequestContext
let channelId = ''
let themeId = ''

async function data<T>(res: Awaited<ReturnType<APIRequestContext['post']>>): Promise<T> {
  expect(res.ok(), await res.text()).toBeTruthy()
  return (await res.json()).data as T
}

test.beforeAll(async () => {
  api = await request.newContext({
    baseURL: `${API_BASE_URL}/`, ignoreHTTPSErrors: true,
    extraHTTPHeaders: { Authorization: `Bearer ${ownerToken()}` },
  })
  channelId = (await data<{ id: string }>(await api.post('integrations/sales-channels', {
    data: { name: `E2E look ${Date.now()}`, type: 'internal', enrollment_mode: 'all', storefront_mode: 'theme' },
  }))).id
  themeId = (await data<{ id: string }>(await api.post('theme/themes', { data: { sales_channel_id: channelId, name: 'Look theme' } }))).id
})

test.afterAll(async () => {
  if (channelId) await api.delete(`integrations/sales-channels/${channelId}`)
  await api.dispose()
})

async function openEditor(page: Page, theme: 'light' | 'dark') {
  const token = (await data<{ token: string }>(await api.post(`theme/themes/${themeId}/sessions`, { data: { purpose: 'edit' } }))).token
  await allowCrossOriginAPI(page)
  await page.goto(`/store/editor#session=${token}&theme=${theme}`)
  await expect(page.getByText('Look theme')).toBeVisible()
  await page.frameLocator('.gjs-frame').locator('section.cos-section').first().waitFor()
}

function luminance(rgb: string): number {
  const [r, g, b] = (rgb.match(/\d+(\.\d+)?/g) ?? []).slice(0, 3).map(Number).map(v => {
    const s = v / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

for (const theme of ['light', 'dark'] as const) {
  test(`the editor chrome in ${theme}`, async ({ page }) => {
    await openEditor(page, theme)
    expect(await page.evaluate(() => document.documentElement.classList.contains('dark'))).toBe(theme === 'dark')
    expect(await page.evaluate(() => getComputedStyle(document.body).fontFamily)).toContain('Atkinson Hyperlegible Next')
    await page.waitForTimeout(800)
    await page.screenshot({ path: `e2e-screenshots/editor-${theme}-add.png` })

    await page.frameLocator('.gjs-frame').locator('section.cos-section h1, section.cos-section h2').first().click()
    await page.getByRole('tab', { name: 'Layers' }).click()
    const selected = page.locator('.gjs-layer.gjs-selected > .gjs-layer-item').first()
    await expect(selected).toBeVisible()
    // The selected layer used to be white on white.
    const [fg, bg] = await selected.evaluate(el => {
      const panel = el.closest('aside') as HTMLElement
      return [getComputedStyle(el).color, getComputedStyle(panel).backgroundColor]
    })
    const [hi, lo] = [luminance(fg), luminance(bg)].sort((a, b) => b - a)
    expect((hi + 0.05) / (lo + 0.05)).toBeGreaterThan(3)
    await page.screenshot({ path: `e2e-screenshots/editor-${theme}-layers-selected.png` })

    await page.getByRole('button', { name: 'Mobile' }).click()
    await page.waitForTimeout(400)
    await page.screenshot({ path: `e2e-screenshots/editor-${theme}-mobile.png` })
  })
}
