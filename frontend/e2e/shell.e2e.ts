import { test, expect } from './fixtures'

// The app shell: grouped sidebar, page titles, account menu, keyboard
// shortcuts, command palette and appearance -- against the real backend.
test.describe('app shell', () => {
  test('sidebar groups navigate and the title follows', async ({ page }) => {
    await page.goto('/')
    const nav = page.getByRole('navigation', { name: 'Main' })

    await nav.getByRole('button', { name: 'Customers' }).click()
    await expect(page).toHaveURL(/\/customers$/)
    await expect(page.getByRole('heading', { level: 1, name: 'Customers' })).toBeVisible()
    await expect(nav.getByRole('button', { name: 'Customers' })).toHaveAttribute('aria-current', 'page')

    const reports = nav.getByRole('button', { name: 'Reports' })
    await expect(reports).toHaveAttribute('aria-expanded', 'false')
    await reports.click()
    await nav.getByRole('button', { name: 'Sales Trends' }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'Reports: Sales Trends' })).toBeVisible()
  })

  test('g then a letter jumps to a page', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeVisible()
    await page.keyboard.press('g')
    await page.keyboard.press('p')
    await expect(page).toHaveURL(/\/products$/)
  })

  test('command palette searches real data and opens a page', async ({ page }) => {
    await page.goto('/')
    await page.keyboard.press('ControlOrMeta+k')
    const search = page.getByRole('combobox', { name: 'Search' })
    await expect(search).toBeFocused()

    await search.fill('supp')
    await expect(page.getByRole('option', { name: /Go to Suppliers/ })).toHaveAttribute('aria-selected', 'true')
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/suppliers$/)
    await expect(page.getByRole('dialog', { name: 'Search' })).toBeHidden()
  })

  test('account menu does not sign out on first click', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'Account: Owner' }).click()
    await expect(page.getByRole('menu')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('menu')).toBeHidden()
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible()
  })

  test('appearance choice survives a reload', async ({ page }) => {
    await page.goto('/')
    const html = page.locator('html')
    await expect(html).not.toHaveClass(/dark/)

    await page.getByRole('button', { name: 'Appearance: Light' }).click()
    await expect(page.getByRole('button', { name: 'Appearance: Dark' })).toBeVisible()
    await expect(html).toHaveClass(/dark/)

    await page.reload()
    await expect(html).toHaveClass(/dark/)
  })
})
