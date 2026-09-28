import { expect, request, test, type APIRequestContext } from '@playwright/test'
import { API_BASE_URL, CONFIGURED, NOT_CONFIGURED_REASON, allowCrossOriginAPI, ownerToken } from './fixtures'

test.skip(!CONFIGURED, NOT_CONFIGURED_REASON)
test.describe.configure({ mode: 'serial' })

let api: APIRequestContext
let channelId = ''
let editToken = ''

async function data<T>(res: Awaited<ReturnType<APIRequestContext['post']>>): Promise<T> {
  expect(res.ok(), await res.text()).toBeTruthy()
  return (await res.json()).data as T
}

test.beforeAll(async () => {
  api = await request.newContext({
    baseURL: `${API_BASE_URL}/`, ignoreHTTPSErrors: true,
    extraHTTPHeaders: { Authorization: `Bearer ${ownerToken()}` },
  })
  const channel = await data<{ id: string }>(await api.post('integrations/sales-channels', {
    data: { name: `E2E store ${Date.now()}`, type: 'internal', enrollment_mode: 'all', storefront_mode: 'theme' },
  }))
  channelId = channel.id
  const theme = await data<{ id: string }>(await api.post('theme/themes', { data: { sales_channel_id: channelId, name: 'E2E theme' } }))
  editToken = (await data<{ token: string }>(await api.post(`theme/themes/${theme.id}/sessions`, { data: { purpose: 'edit' } }))).token
})

test.afterAll(async () => {
  if (channelId) await api.delete(`integrations/sales-channels/${channelId}`)
  await api.dispose()
})

test('a merchant designs a home page in the editor, publishes it, and shoppers see it', async ({ page }) => {
  await allowCrossOriginAPI(page)
  await page.goto(`/store/editor#session=${editToken}`)
  await expect(page.getByText('E2E theme')).toBeVisible()
  expect(page.url()).not.toContain('session=') // token moved out of the address bar

  const canvas = page.frameLocator('.gjs-frame')
  await page.getByText('Section', { exact: true }).dragTo(canvas.locator('body'))
  const section = canvas.locator('section.cos-section')
  await expect(section).toHaveCount(1)
  await page.getByText('Heading', { exact: true }).dragTo(section)
  const heading = canvas.locator('h2.cos-heading')
  await expect(heading).toHaveText('Heading')

  await heading.click()
  const inspector = page.getByRole('complementary', { name: 'Inspector' })
  await inspector.getByLabel('Text').fill('Welcome to the E2E store')
  await expect(heading).toHaveText('Welcome to the E2E store')

  // Undo/redo covers inspector edits.
  await page.getByRole('button', { name: 'Undo' }).click()
  await expect(heading).not.toHaveText('Welcome to the E2E store')
  await page.getByRole('button', { name: 'Redo' }).click()
  await expect(heading).toHaveText('Welcome to the E2E store')

  await page.getByRole('button', { name: 'Publish' }).click()
  await expect(page.getByRole('status')).toContainText('Published version 1')

  const res = await page.goto(`/store/${channelId}/`)
  expect(res?.status()).toBe(200)
  await expect(page.getByRole('heading', { name: 'Welcome to the E2E store' })).toBeVisible()
})

test('unknown store paths render the not-found page with a 404', async ({ page }) => {
  const res = await page.goto(`/store/${channelId}/definitely-not-a-page`)
  expect(res?.status()).toBe(404)
})
