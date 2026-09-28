import { expect, request, test, type APIRequestContext, type Page } from '@playwright/test'
import { API_BASE_URL, CONFIGURED, NOT_CONFIGURED_REASON, allowCrossOriginAPI, ownerToken } from './fixtures'

test.skip(!CONFIGURED, NOT_CONFIGURED_REASON)
test.describe.configure({ mode: 'serial' })

let api: APIRequestContext

/** Drops a block into the canvas the way the browser does: the block's
 *  dragstart, then dragenter/dragover/drop on the target inside the canvas
 *  frame, sharing one DataTransfer. (Playwright's mouse drag doesn't carry
 *  native drag-and-drop across into an iframe.) Target: the first element
 *  matching selector in the canvas. */
async function dragInto(page: Page, blockLabel: string, selector: string, where: 'top' | 'middle') {
  await page.evaluate(async ({ blockLabel, selector, where }) => {
    const block = [...document.querySelectorAll('.gjs-block')].find(b => b.textContent?.trim() === blockLabel)
    const frame = document.querySelector<HTMLIFrameElement>('iframe.gjs-frame')
    const target = frame?.contentDocument?.querySelector(selector)
    if (!block || !target) throw new Error(`drag: missing ${block ? selector : blockLabel}`)
    const r = target.getBoundingClientRect()
    const at = { clientX: r.left + r.width / 2, clientY: where === 'top' ? r.top + 3 : r.top + r.height / 2 }
    const dataTransfer = new DataTransfer()
    const fire = (el: Element, type: string, extra = {}) =>
      el.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer, ...extra }))
    const tick = () => new Promise(r => setTimeout(r, 50))
    fire(block, 'dragstart')
    await tick()
    // The editor places the drop from the dragover positions it has seen.
    for (const type of ['dragenter', 'dragover', 'dragover', 'dragover', 'drop']) {
      fire(target, type, at)
      await tick()
    }
    fire(block, 'dragend')
  }, { blockLabel, selector, where })
}
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
  // Opening a page is not an edit.
  await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeDisabled()

  const canvas = page.frameLocator('.gjs-frame')
  // The default theme's home page already has sections; add one at the top.
  const sections = canvas.locator('section.cos-section')
  await expect(sections.first()).toBeVisible()
  const before = await sections.count()
  await dragInto(page, 'Section', 'section.cos-section', 'top')
  await expect(sections).toHaveCount(before + 1)
  const section = sections.first()
  await dragInto(page, 'Heading', 'section.cos-section', 'middle')
  const heading = section.locator('h2.cos-heading')
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
