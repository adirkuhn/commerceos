import { test, expect } from './fixtures'

// The admin AI assistant against the real stack: a request is queued, its
// own page opens at once and survives a refresh, and cancelling it stops
// it. The local stack has no AI provider connected, so the request stays
// queued (no_provider) -- exactly the "keep it in the queue" behaviour.
test('assistant: ask, land on the queued request, refresh, cancel', async ({ page }) => {
  const question = `e2e ${Date.now()}: what is the default tax setting?`

  await page.goto('/assistant')
  await page.getByLabel('Ask or paste…').fill(question)
  await page.getByRole('button', { name: 'Ask', exact: true }).click()

  await expect(page).toHaveURL(/\/assistant\/[0-9a-f-]{36}$/)
  await expect(page.getByRole('heading', { name: question })).toBeVisible()
  await expect(page.getByText(question).last()).toBeVisible()
  await expect(page.getByText(/Waiting in the queue|The assistant is working on it/)).toBeVisible()

  // Own URL: a refresh shows the same request.
  await page.reload()
  await expect(page.getByRole('heading', { name: question })).toBeVisible()

  await page.getByRole('button', { name: 'Cancel request' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Cancel request' }).click()
  await expect(page.getByText('Cancelled', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Cancel request' })).toHaveCount(0)

  // Back on the list, the request is there, cancelled.
  await page.getByRole('button', { name: 'All requests' }).click()
  await expect(page).toHaveURL(/\/assistant$/)
  const row = page.getByRole('listitem').filter({ hasText: question })
  await expect(row.getByText('Cancelled', { exact: true })).toBeVisible()
})
