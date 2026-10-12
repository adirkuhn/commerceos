import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// index.html reads %VITE_DESIGN_DEFAULT% (rollback switch, see
// src/theme/designPreference). Unset means depot; giving it that value
// here keeps Vite from warning about an undefined variable.
process.env.VITE_DESIGN_DEFAULT ??= 'depot'

export default defineConfig({
  plugins: [react()],
  // OpenAPI contracts (backend/api/*/openapi.yaml) are imported with ?raw by
  // contract tests; without this, Vite's import analysis (used by
  // `vitest related`) tries to parse the YAML as JS and aborts the run.
  assetsInclude: ['**/*.yaml'],
  server: {
    port: 3000,
  },
  test: {
    environment: 'node',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    env: {
      VITE_API_BASE_URL: 'https://api.commerceos.dev.localhost/api/v1',
    },
  },
})
