import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3100,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    // e2e/ is Playwright's own suite (real browser, real backend) --
    // Vitest's default glob would otherwise also pick up its .spec.ts
    // files and collide with Playwright's own test/describe globals.
    exclude: ['**/node_modules/**', 'e2e/**'],
    setupFiles: ['./src/test/setup.ts'],
    env: {
      VITE_API_BASE_URL: 'https://api.commerceos.dev.localhost/api/v1',
      VITE_STOREFRONT_API_KEY: 'test-storefront-key',
    },
  },
})
