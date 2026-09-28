// Storefront server: server-renders shop pages (/store/{channelKey}/...),
// serves the built assets and the GrapesJS editor page. Runs directly on
// Node 24 (native TypeScript type stripping) -- no build step for itself.
import { createServer } from 'node:http'
import { readFile, stat } from 'node:fs/promises'
import { createReadStream } from 'node:fs'
import { join } from 'node:path'
import { createHandler, type HandlerDeps, type RenderInput, type RenderOutput } from './app.ts'
import { staticFileFor } from './static.ts'

const root = join(import.meta.dirname, '..')
const isProd = process.env.NODE_ENV === 'production'
const port = Number(process.env.PORT ?? 3100)
const config = {
  apiInternalURL: process.env.STOREFRONT_API_INTERNAL_URL ?? 'http://localhost:8080/api/v1',
  apiPublicURL: process.env.STOREFRONT_API_PUBLIC_URL ?? 'http://localhost:8080/api/v1',
}

type RenderFn = (input: RenderInput) => Promise<RenderOutput>

async function productionDeps(): Promise<HandlerDeps> {
  const client = join(root, 'dist', 'client')
  const [page, editor] = await Promise.all([readFile(join(client, 'index.html'), 'utf8'), readFile(join(client, 'editor.html'), 'utf8')])
  const entry = join(root, 'dist', 'server', 'entry-server.js')
  const { render } = (await import(entry)) as { render: RenderFn }
  return {
    config, render, pageTemplate: async () => page, editorTemplate: async () => editor,
    serveStatic: async (req, res) => {
      const file = staticFileFor(client, new URL(req.url ?? '/', 'http://x').pathname)
      if (!file || file.path.endsWith('editor.html')) return false
      try {
        await stat(file.path)
      } catch {
        return false
      }
      res.writeHead(200, { 'Content-Type': file.contentType, 'Cache-Control': file.cacheControl, 'X-Content-Type-Options': 'nosniff' })
      createReadStream(file.path).pipe(res)
      return true
    },
  }
}

async function developmentDeps(): Promise<HandlerDeps> {
  const { createServer: createVite } = await import('vite')
  const vite = await createVite({ root, server: { middlewareMode: true }, appType: 'custom' })
  const template = (file: string) => async (url: string) => vite.transformIndexHtml(url, await readFile(join(root, file), 'utf8'))
  return {
    config,
    pageTemplate: template('index.html'),
    editorTemplate: template('editor.html'),
    render: async input => ((await vite.ssrLoadModule('/src/runtime/entry-server.tsx')) as { render: RenderFn }).render(input),
    serveStatic: (req, res) =>
      new Promise(resolve => {
        const { pathname } = new URL(req.url ?? '/', 'http://x')
        // Let Vite serve modules/assets, but not our own page routes.
        if (pathname === '/healthz' || (/^\/store\/[^/]+(\/|$)/.test(pathname) && !/^\/store\/(@|src\/|node_modules\/|assets\/)/.test(pathname))) return resolve(false)
        vite.middlewares(req, res, () => resolve(false))
      }),
  }
}

const deps = isProd ? await productionDeps() : await developmentDeps()
createServer(createHandler(deps)).listen(port, () => {
  console.log(`storefront listening on :${port} (${isProd ? 'production' : 'development'})`)
})
