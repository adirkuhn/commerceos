import { defineConfig, devices } from '@playwright/test'

// E2E runs this admin frontend's dev server (current code) against the
// real local docker-compose backend -- no API mocking, unlike Vitest.
//
// Needs two env vars (see e2e/fixtures.ts):
//   E2E_JWT_SECRET  backend JWT_SECRET  (docker exec commerceos-backend-1 printenv JWT_SECRET)
//   E2E_TENANT_ID   a tenant with data  (select tenant_id from tenant_settings)
// Without them every spec skips with that explanation instead of failing.
const PORT = 3200
const BASE_URL = process.env.E2E_BASE_URL ?? `http://localhost:${PORT}`
export const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'https://api-commerceos.dev.localhost/api/v1'

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  timeout: 30_000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: 'list',
  use: {
    baseURL: BASE_URL,
    // Traefik on the local stack serves a self-signed certificate.
    ignoreHTTPSErrors: true,
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  webServer: {
    command: `npm run dev -- --port ${PORT} --strictPort`,
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    env: {
      VITE_API_BASE_URL: API_BASE_URL,
      VITE_LM2_LOGIN_URL: 'https://lm2.dev.localhost/akantio-pos/app-login',
    },
  },
})
