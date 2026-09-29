import { afterEach, describe, it, expect, vi } from 'vitest'
import { createServer, type Server } from 'node:http'
import type { AddressInfo } from 'node:net'
import { createHandler, type HandlerDeps } from '../app.ts'

let server: Server | undefined
afterEach(() => server?.close())

async function start(overrides: Partial<HandlerDeps> = {}) {
  const deps: HandlerDeps = {
    config: { apiInternalURL: 'http://backend/api/v1', apiPublicURL: 'https://api.test/api/v1' },
    pageTemplate: async () => '<head><!--app-head--></head><body><!--app-html--><script><!--app-state--></script></body>',
    editorTemplate: async () => '<head><!--app-config--></head><body>editor</body>',
    render: vi.fn(async () => ({ status: 200, head: '<title>t</title>', html: '<p>page</p>', state: '{}', lang: 'en' })),
    renderSitemap: vi.fn(async () => ({ status: 200, body: '<urlset/>' })),
    serveStatic: async () => false,
    ...overrides,
  }
  server = createServer(createHandler(deps))
  await new Promise<void>(r => server!.listen(0, r))
  const base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`
  return { deps, get: (path: string, init?: RequestInit) => fetch(base + path, { redirect: 'manual', ...init }) }
}

describe('storefront server', () => {
  it('server-renders store pages with security headers', async () => {
    const { deps, get } = await start()
    const res = await get('/store/key-1/pages/about')

    expect(res.status).toBe(200)
    expect(await res.text()).toBe('<head><title>t</title></head><body><p>page</p><script>{}</script></body>')
    expect(res.headers.get('content-type')).toContain('text/html')
    expect(res.headers.get('x-content-type-options')).toBe('nosniff')
    expect(res.headers.get('x-frame-options')).toBe('SAMEORIGIN')
    expect(deps.render).toHaveBeenCalledWith(expect.objectContaining({
      url: '/store/key-1/pages/about', apiBaseURL: 'http://backend/api/v1', publicApiBaseURL: 'https://api.test/api/v1', previewToken: undefined,
    }))
  })

  it('serves the store sitemap as cacheable XML', async () => {
    const { deps, get } = await start({ config: { apiInternalURL: 'http://backend/api/v1', apiPublicURL: 'https://api.test/api/v1', publicOrigin: 'https://shop.test' } })
    const res = await get('/store/key%201/sitemap.xml')
    expect(res.status).toBe(200)
    expect(await res.text()).toBe('<urlset/>')
    expect(res.headers.get('content-type')).toContain('application/xml')
    expect(res.headers.get('cache-control')).toBe('public, max-age=3600')
    expect(deps.render).not.toHaveBeenCalled()
    expect(deps.renderSitemap).toHaveBeenCalledWith(expect.objectContaining({ storeKey: 'key 1', publicOrigin: 'https://shop.test' }))
  })

  it('404s a sitemap the store cannot have', async () => {
    const { get } = await start({ renderSitemap: async () => ({ status: 404, body: '' }) })
    expect((await get('/store/k/sitemap.xml')).status).toBe(404)
  })

  it('renders through the cached backend fetch', async () => {
    const backend = vi.fn<typeof fetch>()
    const { deps, get } = await start({ fetch: backend })
    await get('/store/key-1/')
    expect(deps.render).toHaveBeenCalledWith(expect.objectContaining({ fetch: backend }))
  })

  it('passes the status through and never caches previews', async () => {
    const render = vi.fn(async () => ({ status: 404, head: '', html: '', state: '', lang: 'en' }))
    const { deps, get } = await start({ render })
    const res = await get('/store/key-1/nope', { headers: { Cookie: 'cos_preview=pv' } })

    expect(res.status).toBe(404)
    expect(res.headers.get('cache-control')).toBe('no-store')
    expect(deps.render).toHaveBeenCalledWith(expect.objectContaining({ previewToken: 'pv' }))
  })

  it('turns a preview link into a cookie and a clean redirect', async () => {
    const { get } = await start()
    const res = await get('/store/key-1/?preview=pv')
    expect(res.status).toBe(302)
    expect(res.headers.get('location')).toBe('/store/key-1/')
    expect(res.headers.get('set-cookie')).toContain('cos_preview=pv')
  })

  it('serves the editor with the public API URL injected, uncached', async () => {
    const { get } = await start()
    const res = await get('/store/editor')
    expect(res.status).toBe(200)
    expect(res.headers.get('cache-control')).toBe('no-store')
    expect(await res.text()).toContain('<meta name="cos-api-base" content="https://api.test/api/v1">')
  })

  it('answers health checks and 404s everything else', async () => {
    const { get } = await start()
    expect((await get('/healthz')).status).toBe(200)
    expect((await get('/admin')).status).toBe(404)
  })

  it('hands static requests to the static server first', async () => {
    const serveStatic = vi.fn(async (_req, res) => { res.end('asset'); return true })
    const { get } = await start({ serveStatic })
    expect(await (await get('/store/assets/x.js')).text()).toBe('asset')
  })

  it('returns a 500 page when rendering throws', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { get } = await start({ render: async () => { throw new Error('boom') } })
    const res = await get('/store/key-1/')
    expect(res.status).toBe(500)
    expect(await res.text()).not.toContain('boom')
    err.mockRestore()
  })
})
