import { renderToString } from 'react-dom/server'
import { QueryClient, QueryClientProvider, dehydrate } from '@tanstack/react-query'
import { StaticRouter } from 'react-router-dom'
import { ApiError, StorefrontAPI } from '../api/storefront'
import type { StorefrontTheme, ThemeDocument } from '../core/document'
import { fontStylesheetURL, settingsToCSS } from '../core/settings'
import { ApiProvider } from './ApiContext'
import { App } from './App'
import { documentQuery, hasDocument, themeQuery } from './queries'
import { serializeState } from './state'
import { documentFor, StoreUnavailable } from './StorePage'

export const BASENAME = '/store'

export interface RenderOptions {
  /** Full request path, e.g. /store/{channelKey}/products/red-shoes */
  url: string
  /** Backend URL the SSR server reaches the API on (internal network). */
  apiBaseURL: string
  /** Backend URL browsers use, handed to the client for hydration. */
  publicApiBaseURL: string
  previewToken?: string
  fetch?: typeof fetch
}

export interface RenderResult {
  status: number
  head: string
  html: string
  state: string
}

const escapeHTML = (s: string) => s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)

function parseStoreURL(url: string): { storeKey: string; path: string } | undefined {
  const pathname = new URL(url, 'http://x').pathname
  const m = /^\/store\/([^/]+)(\/.*)?$/.exec(pathname)
  return m ? { storeKey: decodeURIComponent(m[1]), path: m[2] ?? '/' } : undefined
}

function buildHead(theme: StorefrontTheme, doc: ThemeDocument, preview: boolean): string {
  const title = doc.seo?.title || doc.title || 'Store'
  const parts = [`<title>${escapeHTML(title)}</title>`]
  if (doc.seo?.description) parts.push(`<meta name="description" content="${escapeHTML(doc.seo.description)}">`)
  if (preview) parts.push('<meta name="robots" content="noindex">')
  const fonts = fontStylesheetURL(theme.settings)
  if (fonts) parts.push('<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>', `<link rel="stylesheet" href="${escapeHTML(fonts)}">`)
  parts.push(`<style>${settingsToCSS(theme.settings)}</style>`)
  return parts.join('')
}

const unavailable = (status: number): RenderResult => ({
  status, head: '<title>Store not available</title><meta name="robots" content="noindex">',
  html: renderToString(<StoreUnavailable />), state: '',
})

/**
 * Server-renders one storefront page: loads the theme and the documents
 * the path needs, renders them, and returns the dehydrated cache so the
 * browser hydrates without refetching.
 */
export async function render(opts: RenderOptions): Promise<RenderResult> {
  const parsed = parseStoreURL(opts.url)
  if (!parsed) return unavailable(404)

  const api = new StorefrontAPI({ baseURL: opts.apiBaseURL, storeKey: parsed.storeKey, previewToken: opts.previewToken, fetch: opts.fetch })
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })

  let theme: StorefrontTheme
  try {
    theme = await qc.fetchQuery(themeQuery(api))
  } catch (err) {
    const status = err instanceof ApiError && [401, 404].includes(err.status) ? 404 : 502
    return unavailable(status)
  }

  const route = documentFor(theme, parsed.path)
  const regions = [['section_group', 'header'], ['section_group', 'footer']] as const
  const [main] = await Promise.all([
    qc.fetchQuery(documentQuery(api, theme, route.kind, route.key)),
    ...regions.filter(([k, key]) => hasDocument(theme, k, key)).map(([k, key]) => qc.fetchQuery(documentQuery(api, theme, k, key))),
  ])

  const html = renderToString(
    <QueryClientProvider client={qc}>
      <ApiProvider api={api}>
        <StaticRouter location={new URL(opts.url, 'http://x').pathname} basename={BASENAME}>
          <App />
        </StaticRouter>
      </ApiProvider>
    </QueryClientProvider>,
  )

  return {
    status: route.notFound ? 404 : 200,
    head: buildHead(theme, main, theme.preview || Boolean(opts.previewToken)),
    html,
    state: serializeState({
      dehydrated: dehydrate(qc),
      config: { publicApiBaseURL: opts.publicApiBaseURL, storeKey: parsed.storeKey, previewToken: opts.previewToken },
    }),
  }
}
