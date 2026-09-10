import { defineConfig, devices } from '@playwright/test'

// Unlike storefront-frontend's e2e suite, this one needs no backend, no
// seeded tenant, no env vars -- everything the app talks to is the mock
// repository layer running in the browser itself (see src/repositories).
// `npm run dev` is enough to bring up a fully working stack.
const PORT = 3200
const BASE_URL = process.env.E2E_BASE_URL ?? `http://localhost:${PORT}`

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: 'list',
  use: {
    baseURL: BASE_URL,
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  webServer: {
    command: 'npm run dev',
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
})
