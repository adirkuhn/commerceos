import { renderToString } from 'react-dom/server'
import { QueryClient, QueryClientProvider, dehydrate } from '@tanstack/react-query'
import { StaticRouter } from 'react-router-dom'
import { ApiError, StorefrontAPI } from '../api/storefront'
import type { ReviewList } from '../api/storefront'
import type { StorefrontTheme } from '../core/document'
import { ApiProvider } from '../api/ApiContext'
import { App } from './App'
import { documentQuery, hasDocument, themeQuery } from './queries'
import { dataQueriesFor } from '../components/data'
import { serializeState } from './state'
import { StoreUnavailable } from './StorePage'
import { resolvePageData } from './page/routeData'
import { storeQuery } from './page/queries'
import { buildHead } from './head'
import { canonicalURL } from './seo'
import { sitemapXML } from './sitemap'
import { createI18n } from '../i18n/createI18n'
import { loadShopLocale } from '../i18n/resources'
import { resolveLocale, type Locale } from '../i18n/locale'
import { localeFromCookie, splitLocale, storeLanguages, storePathFor } from './storeLocale'

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
  /** The browser's Accept-Language, for pages without a store (not found). */
  acceptLanguage?: string
  /** Origin shoppers reach the store on (canonical URLs, structured data). */
  publicOrigin?: string
  /** The request's Cookie header (the shopper's remembered language). */
  cookie?: string
}

export interface RenderResult {
  status: number
  head: string
  html: string
  state: string
  /** The page language, for <html lang>. */
  lang: string
  /** Send the shopper here instead (302): their remembered language. */
  redirect?: string
}


function parseStoreURL(url: string): { storeKey: string; path: string } | undefined {
  const pathname = new URL(url, 'http://x').pathname
  const m = /^\/store\/([^/]+)(\/.*)?$/.exec(pathname)
  return m ? { storeKey: decodeURIComponent(m[1]), path: m[2] ?? '/' } : undefined
}

/** The product's review summary, if its reviews were loaded for the page. */
function cachedRating(qc: QueryClient, productId: string): ReviewList['summary'] | undefined {
  const [, reviews] = qc.getQueriesData<ReviewList>({ queryKey: ['reviews', productId] }).find(([, d]) => d) ?? []
  return reviews?.summary
}

async function unavailable(status: number, acceptLanguage?: string): Promise<RenderResult> {
  const locale = resolveLocale(acceptLanguage?.split(',')[0]?.split(';')[0]?.trim())
  await loadShopLocale(locale)
  const title = createI18n(locale).t('store.unavailableTitle')
  return {
    status, lang: locale, head: `<title>${title}</title><meta name="robots" content="noindex">`,
    html: renderToString(<StoreUnavailable locale={locale} />), state: '',
  }
}

/**
 * Server-renders one storefront page: loads the theme and the documents
 * the path needs, renders them, and returns the dehydrated cache so the
 * browser hydrates without refetching.
 */
export async function render(opts: RenderOptions): Promise<RenderResult> {
  const parsed = parseStoreURL(opts.url)
  if (!parsed) return unavailable(404, opts.acceptLanguage)

  const api = new StorefrontAPI({ baseURL: opts.apiBaseURL, storeKey: parsed.storeKey, previewToken: opts.previewToken, fetch: opts.fetch })
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })

  let theme: StorefrontTheme
  try {
    theme = await qc.fetchQuery(themeQuery(api))
  } catch (err) {
    const status = err instanceof ApiError && [401, 404].includes(err.status) ? 404 : 502
    return unavailable(status, opts.acceptLanguage)
  }

  const requested = new URL(opts.url, 'http://x')
  const query = requested.searchParams
  const store = await qc.fetchQuery(storeQuery(api))
  const langs = storeLanguages(store)
  const { locale, path, prefixed } = splitLocale(parsed.path, langs)
  const remembered = prefixed ? undefined : localeFromCookie(opts.cookie, langs)
  if (remembered && remembered !== locale) {
    return { status: 302, redirect: storePathFor(parsed.storeKey, remembered, langs)(path.replace(/^\/+/, '') + requested.search), head: '', html: '', state: '', lang: remembered }
  }

  const page = await resolvePageData(qc, api, theme, path, query)
  const route = page.route
  const regions = [['section_group', 'header'], ['section_group', 'footer']] as const
  const [main, regionDocs] = await Promise.all([
    route.builtin ? undefined : qc.fetchQuery(documentQuery(api, theme, route.kind, route.key)),
    Promise.all(regions.filter(([k, key]) => hasDocument(theme, k, key)).map(([k, key]) => qc.fetchQuery(documentQuery(api, theme, k, key)))),
  ])
  const docs = main ? [main, ...regionDocs] : regionDocs
  // The page renders synchronously: its language's texts must be loaded first.
  await loadShopLocale(locale)
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

  // A page's address in one language: /{lang}/… unless it is the default.
  const localized = (l: Locale, p: string) => (l === langs.defaultLocale ? p : `/${l}${p}`)
  const urlIn = (l: Locale) => canonicalURL(opts.publicOrigin, parsed.storeKey, localized(l, path), query, route)
  const alternates = opts.publicOrigin
    ? [...langs.offered.map(l => ({ hreflang: l, href: urlIn(l)! })), { hreflang: 'x-default', href: urlIn(langs.defaultLocale)! }]
    : undefined
  return {
    status: route.notFound ? 404 : 200,
    lang: locale,
    head: buildHead({
      theme, doc: main, page, preview: theme.preview || Boolean(opts.previewToken), storeName: store?.name ?? '',
      canonical: urlIn(locale),
      homeURL: canonicalURL(opts.publicOrigin, parsed.storeKey, localized(locale, '/'), new URLSearchParams(), route),
      alternates,
      rating: page.product ? cachedRating(qc, page.product.id) : undefined,
    }),
    html,
    state: serializeState({
      dehydrated: dehydrate(qc),
      config: { publicApiBaseURL: opts.publicApiBaseURL, storeKey: parsed.storeKey, previewToken: opts.previewToken },
    }),
  }
}

export interface SitemapOptions {
  storeKey: string
  apiBaseURL: string
  publicOrigin?: string
  fetch?: typeof fetch
}

/** The live store's sitemap.xml; 404 without a public origin (sitemaps
 *  need absolute URLs) or when the store isn't available. */
export async function renderSitemap(opts: SitemapOptions): Promise<{ status: number; body: string }> {
  if (!opts.publicOrigin) return { status: 404, body: '' }
  const api = new StorefrontAPI({ baseURL: opts.apiBaseURL, storeKey: opts.storeKey, fetch: opts.fetch })
  try {
    const [theme, data, store] = await Promise.all([api.getTheme(), api.getSitemap(), api.getStore().catch(() => null)])
    return { status: 200, body: sitemapXML(opts.publicOrigin, opts.storeKey, theme, data, storeLanguages(store)) }
  } catch (err) {
    if (err instanceof ApiError && [401, 404, 422].includes(err.status)) return { status: 404, body: '' }
    throw err
  }
}

