import { defineConfig, devices } from '@playwright/test'

// E2E runs the real storefront-frontend dev server against a real backend
// -- there is no mocking here, unlike the Vitest unit suite. It needs a
// seeded tenant: an internal sales channel whose currency_code is set to
// E2E_EXPECTED_CURRENCY (default "USD"), an fx.ExchangeRate configured for
// it (or per-SKU/coupon overrides), at least one active product enrolled
// and in stock, and a storefront_api_keys row scoped to that channel --
// see e2e/multi-currency-checkout.spec.ts's own doc comment for the exact
// env vars this expects. Without them, the spec explains itself and skips
// rather than failing on a red herring.
const PORT = 3100
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
    // The local stack's Traefik edge terminates TLS with a self-signed
    // cert (see project deploy notes) -- browsers reject it by default.
    ignoreHTTPSErrors: true,
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
    env: {
      VITE_API_BASE_URL: process.env.VITE_API_BASE_URL ?? 'https://api.commerceos.dev.localhost/api/v1',
      VITE_STOREFRONT_API_KEY: process.env.E2E_STOREFRONT_API_KEY ?? '',
    },
  },
})
