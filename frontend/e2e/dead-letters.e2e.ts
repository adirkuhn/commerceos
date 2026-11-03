import { randomUUID } from 'node:crypto'
import { test as base, expect, mintToken, API_BASE_URL, CONFIGURED, NOT_CONFIGURED_REASON } from './fixtures'
import { publish } from './nats'

// The super-admin dead-letter page (COM-14) against the real stack: a dead
// letter is put on the events DLQ the way a failing consumer would, then
// inspected, retried and discarded from the UI.
//
// The dead letters name a consumer no subscription has ("e2e-<stamp>"), so
// a retry is skipped by every real consumer -- nothing in the tenant runs.
const test = base.extend({ token: async ({}, use) => { await use(mintToken('super_admin')) } })

const SUBJECT = 'oms.sales_order.status_changed'
const TENANT_ID = process.env.E2E_TENANT_ID ?? ''

async function deadLetter(consumer: string): Promise<string> {
  const eventId = randomUUID()
  await publish(`dlq.${SUBJECT}`, JSON.stringify({ event_id: eventId, tenant_id: TENANT_ID, sales_order_id: randomUUID() }), {
    'Commerceos-Consumer': consumer,
    'Commerceos-Original-Subject': SUBJECT,
    'Commerceos-Error': 'woo: 403 forbidden (e2e)',
    'Commerceos-Num-Delivered': '5',
    'Commerceos-Dead-Lettered-At': new Date().toISOString().replace(/\.\d+Z$/, 'Z'),
  })
  return eventId
}

test('dead letters: inspect one, retry it for its consumer, discard another', async ({ page }) => {
  const consumer = `e2e-${Date.now()}`
  const retried = await deadLetter(consumer)
  const discarded = await deadLetter(consumer)

  await page.goto(`/dead-letters?consumer=${consumer}`)
  const rows = page.getByRole('row').filter({ hasText: consumer })
  await expect(rows).toHaveCount(2)
  await expect(page.getByTestId('count-events')).not.toHaveText('0')

  // Detail: failure details + the event data, pretty-printed.
  await rows.first().getByRole('link').click()
  await expect(page).toHaveURL(/\/dead-letters\/events\/\d+$/)
  await expect(page.getByText('woo: 403 forbidden (e2e)')).toBeVisible()
  await expect(page.getByTestId('dead-letter-data')).toContainText(`"event_id": "${retried}"`)
  await expect(page.getByText(/re-read the current state/).first()).toBeVisible()

  // Refresh keeps the dead letter open (own URL).
  await page.reload()
  await expect(page.getByTestId('dead-letter-data')).toContainText(retried)

  await page.getByRole('button', { name: 'Retry', exact: true }).click()
  const retryDialog = page.getByRole('alertdialog')
  await expect(retryDialog).toContainText(`for ${consumer} only`)
  await retryDialog.getByRole('button', { name: 'Retry event' }).click()

  await expect(page).toHaveURL(/\/dead-letters\?/)
  await expect(rows).toHaveCount(1)

  await rows.first().getByRole('link').click()
  await expect(page.getByTestId('dead-letter-data')).toContainText(discarded)
  await page.getByRole('button', { name: 'Discard', exact: true }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Discard event' }).click()

  await expect(page).toHaveURL(/\/dead-letters\?/)
  await expect(page.getByText('No dead letters')).toBeVisible()
})

test('dead letters: only a super-admin reaches the API', async ({ request }) => {
  test.skip(!CONFIGURED, NOT_CONFIGURED_REASON)
  for (const role of ['customer_owner', 'customer_admin', 'customer_user']) {
    const res = await request.get(`${API_BASE_URL}/admin/dead-letters?stream=events`, { headers: { Authorization: `Bearer ${mintToken(role)}` } })
    expect(res.status(), role).toBe(403)
  }
  const ok = await request.get(`${API_BASE_URL}/admin/dead-letters?stream=events`, { headers: { Authorization: `Bearer ${mintToken('super_admin')}` } })
  expect(ok.status()).toBe(200)
})
