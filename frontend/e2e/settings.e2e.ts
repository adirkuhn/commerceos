import { test, expect, API_BASE_URL } from './fixtures'

// Company profile and notification preferences against the real backend.
// Needs a backend built from this commit or later (older builds have no
// /settings/company-profile) -- skips with that reason otherwise.
test.beforeEach(async ({ request, token }) => {
  const res = await request.get(`${API_BASE_URL}/settings/company-profile`, { headers: { Authorization: `Bearer ${token}` } })
  test.skip(res.status() === 404, 'backend predates the company-profile API -- rebuild the local stack (docker compose build backend scheduler)')
})

test('company profile: save, reload, and it is still there', async ({ page }) => {
  const name = `E2E Company ${Date.now()} B.V.`
  await page.goto('/settings')

  const legal = page.getByLabel('Legal name')
  await legal.fill(name)
  await page.getByLabel('VAT number').fill('NL123456789B01')
  await page.getByLabel('Country').selectOption('NL')
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(page.getByRole('status')).toContainText('Saved')

  await page.reload()
  await expect(page.getByLabel('Legal name')).toHaveValue(name)
  await expect(page.getByRole('button', { name: 'Save changes' })).toBeDisabled()
})

test('company profile: an invalid IBAN is shown on the IBAN field', async ({ page }) => {
  await page.goto('/settings')
  await page.getByLabel('Legal name').fill('E2E B.V.')
  const iban = page.getByLabel('IBAN')
  await iban.fill('NL00 BAD')
  await page.getByRole('button', { name: 'Save changes' }).click()

  await expect(iban).toHaveAttribute('aria-invalid', 'true')
  await expect(iban).toBeFocused()
})

test('notifications: email without recipients is refused on the recipients field', async ({ page }) => {
  await page.goto('/settings/notifications')
  await page.getByRole('checkbox', { name: 'Email imports' }).check()
  await page.getByLabel('Email recipients').fill('')
  await page.getByRole('button', { name: 'Save changes' }).click()

  await expect(page.getByLabel('Email recipients')).toHaveAttribute('aria-invalid', 'true')
})
