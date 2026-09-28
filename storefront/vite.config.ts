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
