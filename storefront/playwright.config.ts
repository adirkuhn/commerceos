import { defineConfig, devices } from '@playwright/test'

// E2E runs this storefront's server (current code, Vite dev mode) against
// the real local docker-compose backend. Same env as the admin E2E suite:
//   E2E_JWT_SECRET  backend JWT_SECRET
//   E2E_TENANT_ID   a tenant_settings.tenant_id
// Without them every spec skips with that explanation.
const PORT = 3300
const BASE_URL = process.env.E2E_STOREFRONT_URL ?? `http://localhost:${PORT}`
const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'https://api-commerceos.dev.localhost/api/v1'

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  timeout: 60_000,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: 'list',
  use: { baseURL: BASE_URL, ignoreHTTPSErrors: true, trace: 'retain-on-failure' },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } }],
  webServer: {
    command: 'node server/main.ts',
    url: `${BASE_URL}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    env: {
      PORT: String(PORT),
      STOREFRONT_API_INTERNAL_URL: API_BASE_URL,
      STOREFRONT_API_PUBLIC_URL: API_BASE_URL,
      // Canonical URLs and the sitemap point at this server.
      STOREFRONT_PUBLIC_URL: BASE_URL,
      // The local stack's Traefik certificate is self-signed.
      NODE_TLS_REJECT_UNAUTHORIZED: '0',
    },
  },
})
