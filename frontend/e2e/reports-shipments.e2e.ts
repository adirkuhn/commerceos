import { test, expect } from './fixtures'

// Shipments report: grouping and basis switch the counts and live in the
// URL, so a refresh shows the same report.
test('reports: shipments per week/month/quarter/year', async ({ page }) => {
  await page.goto('/reports/shipments')
  await expect(page.getByRole('heading', { name: /Shipments/ })).toBeVisible()
  await expect(page.getByTestId('total-shipments')).toBeVisible()

  const groupBy = page.getByRole('group', { name: 'Group by' })
  await groupBy.getByRole('button', { name: 'Quarter' }).click()
  await expect(page).toHaveURL(/granularity=quarter/)
  await expect(page.getByText('Shipments per quarter')).toBeVisible()

  await page.getByRole('button', { name: 'Carrier scan' }).click()
  await expect(page).toHaveURL(/basis=carrier_scan/)

  await page.reload()
  await expect(groupBy.getByRole('button', { name: 'Quarter' })).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('button', { name: 'Carrier scan' })).toHaveAttribute('aria-pressed', 'true')
})
