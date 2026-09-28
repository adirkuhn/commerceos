import { resolve } from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Two browser entries: the shopper runtime (index.html) and the GrapesJS
// editor (editor.html). They share src/core and src/components; only the
// editor entry imports grapesjs. The SSR bundle is built separately
// (see package.json "build").
export default defineConfig({
  base: '/store/',
  plugins: [react()],
  ssr: {
    // Resolve external packages the way Node 24 does in production: its
    // "module-sync" condition picks react-router's ESM build. Without it
    // dev SSR loads the CJS build, whose re-exports (StaticRouter) Vite
    // can't see as named exports and every page fails to render.
    resolve: { externalConditions: ['module-sync'] },
  },
  build: {
    rollupOptions: {
      input: {
        runtime: resolve(import.meta.dirname, 'index.html'),
        editor: resolve(import.meta.dirname, 'editor.html'),
      },
    },
  },
  test: {
    environment: 'node',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    exclude: ['e2e/**', 'node_modules/**'],
  },
})
