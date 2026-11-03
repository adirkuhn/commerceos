import { test, expect, api } from './fixtures'

interface Connection { id: string; provider: string }
interface SyncSchedule { entity_type: string; direction: string; enabled: boolean; interval_minutes: number }

// COM-18: the WooCommerce Products and Stock schedules are gone (changes
// reach the store through events), and Stock reconciliation -- export
// only, daily by default -- takes their place as the safety net. Saved
// through the real API, so the contract (entity type enum) and the
// validation both run.
test('woo sync settings: no products/stock schedules, stock reconciliation saves as a daily export', async ({ page, request, token }) => {
  const connections = await api<Connection[]>(request, token, '/integrations/connections', { method: 'GET' })
  const woo = connections.find(c => c.provider === 'woocommerce')
  test.skip(!woo, 'tenant needs a WooCommerce connection')
  const before = await api<SyncSchedule[]>(request, token, `/integrations/connections/${woo!.id}/sync-schedules`, { method: 'GET' })
  const previous = before.find(s => s.entity_type === 'stock_reconciliation' && s.direction === 'export')

  try {
    await page.goto(`/settings/integrations/${woo!.id}`)
    await expect(page.getByText('Sync schedule')).toBeVisible()
    await expect(page.getByRole('switch', { name: 'Products import' })).toHaveCount(0)
    await expect(page.getByRole('switch', { name: 'Stock levels export' })).toHaveCount(0)
    await expect(page.getByRole('switch', { name: 'Stock reconciliation import' })).toBeDisabled()

    const exportSwitch = page.getByRole('switch', { name: 'Stock reconciliation export' })
    if (!(await exportSwitch.isChecked())) await exportSwitch.click({ force: true })
    await page.getByRole('combobox', { name: 'Stock reconciliation export interval' }).selectOption('1440')
    await page.getByRole('button', { name: 'Save', exact: true }).click()

    await expect.poll(async () => {
      const after = await api<SyncSchedule[]>(request, token, `/integrations/connections/${woo!.id}/sync-schedules`, { method: 'GET' })
      const row = after.find(s => s.entity_type === 'stock_reconciliation' && s.direction === 'export')
      return row && { enabled: row.enabled, interval: row.interval_minutes }
    }).toEqual({ enabled: true, interval: 1440 })
    const after = await api<SyncSchedule[]>(request, token, `/integrations/connections/${woo!.id}/sync-schedules`, { method: 'GET' })
    expect(after.filter(s => s.entity_type === 'products' || s.entity_type === 'stock')).toEqual([])
  } finally {
    // put the tenant's own choice back
    if (!previous?.enabled) {
      const after = await api<SyncSchedule[]>(request, token, `/integrations/connections/${woo!.id}/sync-schedules`, { method: 'GET' }).catch(() => [])
      await api(request, token, `/integrations/connections/${woo!.id}/sync-schedules`, {
        method: 'PUT', body: { schedules: after.map(s => s.entity_type === 'stock_reconciliation' ? { ...s, enabled: false } : s) },
      }).catch(() => {})
    }
  }
})
