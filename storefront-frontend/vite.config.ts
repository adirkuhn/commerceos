import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ command }) => ({
  plugins: [react()],
  // Production is deployed path-mounted at /store on the same host/domain
  // the backend API already has DNS/TLS for (see docker-compose.yml's
  // storefront service) -- no separate domain to provision just to get a
  // working preview URL. Only affects the build (asset URLs the emitted
  // index.html references); `npm run dev`/tests keep serving from root so
  // local dev isn't forced through that same path.
  base: command === 'build' ? '/store/' : '/',
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
}))
