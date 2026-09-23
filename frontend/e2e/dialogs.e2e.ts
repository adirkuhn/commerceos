import { test, expect, api } from './fixtures'

interface Supplier { id: string; name: string }

// In-app confirmation and modal dialog behaviour on real records.
test.describe('dialogs', () => {
  test('deleting a supplier asks first; Cancel keeps it, confirming deletes it', async ({ page, request, token }) => {
    const name = `E2E supplier ${Date.now()}`
    const supplier = await api<Supplier>(request, token, '/suppliers', { method: 'POST', body: { name, status: 'active' } })
    try {
      await page.goto('/suppliers')
      const row = page.getByRole('row', { name: new RegExp(name) })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Delete' }).click()
      const confirm = page.getByRole('alertdialog', { name: `Delete supplier "${name}"?` })
      await expect(confirm).toBeVisible()
      await confirm.getByRole('button', { name: 'Cancel' }).click()
      await expect(confirm).toBeHidden()
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Delete' }).click()
      await page.keyboard.press('Enter') // no default button: Return must not confirm
      await expect(confirm).toBeVisible()
      await confirm.getByRole('button', { name: 'Delete supplier' }).click()
      await expect(row).toBeHidden()
    } finally {
      await api(request, token, `/suppliers/${supplier.id}`, { method: 'DELETE' }).catch(() => undefined)
    }
  })

  test('a modal traps focus, closes on Escape and returns focus to its opener', async ({ page }) => {
    await page.goto('/integrations')
    const opener = page.getByRole('button', { name: /Browse integrations/ }).first()
    await opener.click()

    const dialog = page.getByRole('dialog', { name: 'Browse integrations' })
    await expect(dialog).toBeVisible()
    for (let i = 0; i < 8; i++) {
      await page.keyboard.press('Tab')
      await expect(dialog.locator(':focus')).toHaveCount(1)
    }

    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await expect(opener).toBeFocused()
  })
})
