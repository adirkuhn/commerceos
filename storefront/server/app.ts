import type { IncomingMessage, ServerResponse } from 'node:http'
import { injectPage } from './html.ts'
import { previewRedirect, previewTokenFrom } from './preview.ts'

export interface RenderInput {
  url: string
  apiBaseURL: string
  publicApiBaseURL: string
  previewToken?: string
  fetch?: typeof fetch
  publicOrigin?: string
  acceptLanguage?: string
  cookie?: string
}

export interface RenderOutput {
  status: number
  head: string
  html: string
  state: string
  lang: string
  redirect?: string
}

export interface SitemapInput {
  storeKey: string
  apiBaseURL: string
  publicOrigin?: string
  fetch?: typeof fetch
}

export interface HandlerDeps {
  /** publicOrigin: where shoppers reach /store (canonical URLs); optional. */
  config: { apiInternalURL: string; apiPublicURL: string; publicOrigin?: string }
  /** index.html (dev: transformed by Vite for the request URL). */
  pageTemplate: (url: string) => Promise<string>
  editorTemplate: (url: string) => Promise<string>
  render: (input: RenderInput) => Promise<RenderOutput>
  renderSitemap: (input: SitemapInput) => Promise<{ status: number; body: string }>
  /** How the renderer reaches the backend (cached, see httpCache.ts). */
  fetch?: typeof fetch
  /** Serves built/dev assets; resolves true when it handled the request. */
  serveStatic: (req: IncomingMessage, res: ServerResponse) => Promise<boolean>
}

const SECURITY_HEADERS: Record<string, string> = {
  'X-Content-Type-Options': 'nosniff',
  'X-Frame-Options': 'SAMEORIGIN',
  'Referrer-Policy': 'strict-origin-when-cross-origin',
}

const escapeAttr = (s: string) => s.replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]!)

function send(res: ServerResponse, status: number, body: string, headers: Record<string, string>) {
  res.writeHead(status, { ...SECURITY_HEADERS, ...headers })
  res.end(body)
}

export function createHandler(deps: HandlerDeps) {
  return async (req: IncomingMessage, res: ServerResponse) => {
    const url = req.url ?? '/'
    const { pathname } = new URL(url, 'http://x')
    try {
      if (pathname === '/healthz') return send(res, 200, 'ok', { 'Content-Type': 'text/plain' })

      const preview = previewRedirect(url)
      if (preview) return send(res, 302, '', { Location: preview.location, 'Set-Cookie': preview.cookie, 'Cache-Control': 'no-store' })

      if (await deps.serveStatic(req, res)) return

      if (pathname === '/store/editor' || pathname === '/store/editor/') {
        const html = (await deps.editorTemplate(url)).replace('<!--app-config-->', () =>
          `<meta name="cos-api-base" content="${escapeAttr(deps.config.apiPublicURL)}">`)
        return send(res, 200, html, { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store' })
      }

      const sitemap = /^\/store\/([^/]+)\/sitemap\.xml$/.exec(pathname)
      if (sitemap) {
        const out = await deps.renderSitemap({
          storeKey: decodeURIComponent(sitemap[1]), apiBaseURL: deps.config.apiInternalURL, publicOrigin: deps.config.publicOrigin, fetch: deps.fetch,
        })
        return out.status === 200
          ? send(res, 200, out.body, { 'Content-Type': 'application/xml; charset=utf-8', 'Cache-Control': 'public, max-age=3600' })
          : send(res, out.status, 'Not found', { 'Content-Type': 'text/plain' })
      }

      if (pathname.startsWith('/store/')) {
        const previewToken = previewTokenFrom(req.headers.cookie)
        const page = await deps.render({
          url, apiBaseURL: deps.config.apiInternalURL, publicApiBaseURL: deps.config.apiPublicURL, previewToken, fetch: deps.fetch,
          publicOrigin: deps.config.publicOrigin, acceptLanguage: req.headers['accept-language'], cookie: req.headers.cookie,
        })
        if (page.redirect) return send(res, 302, '', { Location: page.redirect, 'Cache-Control': 'no-store', Vary: 'Cookie' })
        const html = injectPage(await deps.pageTemplate(url), page)
        return send(res, page.status, html, {
          'Content-Type': 'text/html; charset=utf-8',
          // Pages are rendered per request; theme data is cached upstream.
          'Cache-Control': previewToken ? 'no-store' : 'no-cache',
          // Plain URLs open in the shopper's remembered language.
          Vary: 'Cookie',
        })
      }

      return send(res, 404, 'Not found', { 'Content-Type': 'text/plain' })
    } catch (err) {
      console.error('storefront: request failed', url, err)
      if (!res.headersSent) send(res, 500, '<!doctype html><title>Error</title><p>Something went wrong. Please try again.</p>', { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store' })
    }
  }
}
