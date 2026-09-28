import { renderToString } from 'react-dom/server'
import { QueryClient, QueryClientProvider, dehydrate } from '@tanstack/react-query'
import { StaticRouter } from 'react-router-dom'
import { ApiError, StorefrontAPI } from '../api/storefront'
import type { StorefrontTheme, ThemeDocument } from '../core/document'
import { fontStylesheetURL, settingsToCSS } from '../core/settings'
import { ApiProvider } from '../api/ApiContext'
import { App } from './App'
import { documentQuery, hasDocument, themeQuery } from './queries'
import { dataQueriesFor } from '../components/data'
import { serializeState } from './state'
import { StoreUnavailable } from './StorePage'
import { resolvePageData, type PageData } from './page/routeData'
import { storeQuery } from './page/queries'

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

function buildHead(theme: StorefrontTheme, doc: ThemeDocument | undefined, page: PageData, preview: boolean): string {
  // Built-in pages carry one-time tokens in their URL: keep them out of
  // search results and out of Referer headers.
  const noindex = preview || Boolean(page.route.builtin)
  const title = page.seoTitle || doc?.seo?.title || doc?.title || 'Store'
  const description = page.seoDescription || doc?.seo?.description
  const parts = [`<title>${escapeHTML(title)}</title>`]
  if (description) parts.push(`<meta name="description" content="${escapeHTML(description)}">`)
  if (noindex) parts.push('<meta name="robots" content="noindex">')
  if (page.route.builtin) parts.push('<meta name="referrer" content="no-referrer">')
  const favicon = theme.settings.brand.favicon_asset_id && theme.assets?.[theme.settings.brand.favicon_asset_id]
  if (favicon) parts.push(`<link rel="icon" href="${escapeHTML(favicon.url)}">`)
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

  const page = await resolvePageData(qc, api, theme, parsed.path, new URL(opts.url, 'http://x').searchParams)
  const route = page.route
  const regions = [['section_group', 'header'], ['section_group', 'footer']] as const
  const [main, regionDocs] = await Promise.all([
    route.builtin ? undefined : qc.fetchQuery(documentQuery(api, theme, route.kind, route.key)),
    Promise.all(regions.filter(([k, key]) => hasDocument(theme, k, key)).map(([k, key]) => qc.fetchQuery(documentQuery(api, theme, k, key)))),
    qc.fetchQuery(storeQuery(api)),
  ])
  const docs = main ? [main, ...regionDocs] : regionDocs
  // Catalog data the components show. A failed fetch is left out of the
  // cache; that component renders nothing rather than failing the page.
  await Promise.all(docs.flatMap(d => dataQueriesFor(d.root, api, { product: page.product })).map(q => qc.prefetchQuery(q)))

  const html = renderToString(
    <QueryClientProvider client={qc}>
      <ApiProvider api={api}>
        <StaticRouter location={((u: URL) => u.pathname + u.search)(new URL(opts.url, 'http://x'))} basename={BASENAME}>
          <App />
        </StaticRouter>
      </ApiProvider>
    </QueryClientProvider>,
  )

  return {
    status: route.notFound ? 404 : 200,
    head: buildHead(theme, main, page, theme.preview || Boolean(opts.previewToken)),
    html,
    state: serializeState({
      dehydrated: dehydrate(qc),
      config: { publicApiBaseURL: opts.publicApiBaseURL, storeKey: parsed.storeKey, previewToken: opts.previewToken },
    }),
  }
}
