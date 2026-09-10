import { test, expect } from '@playwright/test'

test('logs in with the seeded demo account and reaches the account dashboard', async ({ page }) => {
  await page.goto('/login')

  await page.getByLabel('Email', { exact: true }).fill('morgan@example.com')
  await page.getByLabel('Password').fill('demo-password')
  await page.getByRole('button', { name: 'Log in' }).click()

  await expect(page).toHaveURL(/\/account$/)
  await expect(page.getByRole('heading', { name: /Welcome back, Morgan/ })).toBeVisible()
})

test('shows an error for an incorrect password and stays on the login page', async ({ page }) => {
  await page.goto('/login')

  await page.getByLabel('Email', { exact: true }).fill('morgan@example.com')
  await page.getByLabel('Password').fill('wrong-password')
  await page.getByRole('button', { name: 'Log in' }).click()

  await expect(page.getByRole('alert')).toHaveText('Incorrect email or password.')
  await expect(page).toHaveURL(/\/login$/)
})

test('redirects an unauthenticated visitor from /account to /login', async ({ page }) => {
  await page.goto('/account')
  await expect(page).toHaveURL(/\/login/)
})
