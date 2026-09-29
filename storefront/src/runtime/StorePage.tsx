import { useSuspenseQuery } from '@tanstack/react-query'
import { useMemo, type ReactNode } from 'react'
import { I18nextProvider, useTranslation } from 'react-i18next'
import { createI18n } from '../i18n/createI18n'
import { resolveLocale, type Locale } from '../i18n/locale'
import { useStorefrontAPI } from '../api/ApiContext'
import type { StorefrontTheme } from '../core/document'
import type { Collection } from '../api/storefront'
import { DocumentRenderer } from './DocumentRenderer'
import { RenderProvider } from './RenderContext'
import { documentQuery, documentFor, hasDocument, themeQuery } from './queries'
import type { ResolvedRoute } from './route'
import { CartProvider } from './page/CartContext'
import { CustomerProvider } from './page/CustomerContext'
import { ListingProvider } from './page/ListingContext'
import { OrderProvider } from './page/OrderContext'
import { ProductProvider } from './page/ProductContext'
import { StoreInfoProvider } from './page/StoreContext'
import { WishlistProvider } from './page/WishlistContext'
import { collectionPageQuery, listingQuery, productPageQuery, storeQuery } from './page/queries'
import { NewsletterPage } from './builtin/NewsletterPage'
import { StoreLanguageProvider } from './page/StoreLanguageContext'
import { splitLocale, storeLanguages, storePathFor } from './storeLocale'
import { listingParamsFor } from './page/routeData'

export { documentFor }

const NOT_FOUND: ResolvedRoute = { kind: 'template', key: 'not-found', notFound: true }

function Region({ theme, kind, docKey }: { theme: StorefrontTheme; kind: ResolvedRoute['kind']; docKey: string }) {
  const api = useStorefrontAPI()
  const { data } = useSuspenseQuery(documentQuery(api, theme, kind, docKey))
  return <DocumentRenderer root={data.root} assets={data.assets ?? {}} />
}

function Main({ theme, route }: { theme: StorefrontTheme; route: ResolvedRoute }) {
  switch (route.builtin) {
    case 'newsletter-confirm': return <NewsletterPage action="confirm" />
    case 'newsletter-unsubscribe': return <NewsletterPage action="unsubscribe" />
    default: return <Region theme={theme} kind={route.kind} docKey={route.key} />
  }
}

/** Header, the route's document, footer. */
function Layout({ theme, route }: { theme: StorefrontTheme; route: ResolvedRoute }) {
  const has = (key: string) => hasDocument(theme, 'section_group', key)
  return (
    <>
      {has('header') && <header className="cos-header"><Region theme={theme} kind="section_group" docKey="header" /></header>}
      <main className="cos-main"><Main theme={theme} route={route} /></main>
      {has('footer') && <footer className="cos-footer"><Region theme={theme} kind="section_group" docKey="footer" /></footer>}
    </>
  )
}

function ProductRoute({ theme, route }: { theme: StorefrontTheme; route: ResolvedRoute }) {
  const api = useStorefrontAPI()
  const { data: product } = useSuspenseQuery(productPageQuery(api, route.param ?? ''))
  if (!product) return <Layout theme={theme} route={NOT_FOUND} />
  return <ProductProvider product={product}><Layout theme={theme} route={route} /></ProductProvider>
}

interface ListingRouteProps {
  theme: StorefrontTheme
  route: ResolvedRoute
  query: URLSearchParams
  storePath: (p: string) => string
}

function ListingBody({ theme, route, query, storePath, collection }: ListingRouteProps & { collection: Collection | null }) {
  const api = useStorefrontAPI()
  const { t } = useTranslation()
  const { data: doc } = useSuspenseQuery(documentQuery(api, theme, route.kind, route.key))
  const params = listingParamsFor(route, query, doc, collection)
  const { data: listing } = useSuspenseQuery(listingQuery(api, params))
  const base = collection ? `collections/${encodeURIComponent(route.param ?? '')}` : 'search'
  const href = (changes: Record<string, string | null>) => {
    const next = new URLSearchParams(query)
    for (const [k, v] of Object.entries(changes)) {
      if (v === null) next.delete(k)
      else next.set(k, v)
    }
    const qs = next.toString()
    return storePath(qs ? `${base}?${qs}` : base)
  }
  return (
    <ListingProvider value={{
      title: collection ? collection.name : t('listing.searchResults', { query: params.search ?? '' }),
      description: collection?.description ?? '',
      imageUrl: collection?.image_url,
      listing, page: params.page,
      pageCount: Math.max(1, Math.ceil(listing.total / params.perPage)),
      activeCategoryId: params.categoryId,
      hrefForPage: p => href({ page: p > 1 ? String(p) : null }),
      hrefForCategory: id => href({ category: id, page: null }),
    }}>
      <Layout theme={theme} route={route} />
    </ListingProvider>
  )
}

function CollectionRoute(props: ListingRouteProps) {
  const api = useStorefrontAPI()
  const { data: collection } = useSuspenseQuery(collectionPageQuery(api, props.route.param ?? ''))
  if (!collection) return <Layout theme={props.theme} route={NOT_FOUND} />
  return <ListingBody {...props} collection={collection} />
}

function Routed({ theme, route, query, storePath }: { theme: StorefrontTheme; route: ResolvedRoute; query: URLSearchParams; storePath: (p: string) => string }): ReactNode {
  if (route.key === 'product' && !route.notFound) return <ProductRoute theme={theme} route={route} />
  if (route.key === 'collection' && !route.notFound) return <CollectionRoute theme={theme} route={route} query={query} storePath={storePath} />
  if (route.key === 'search' && !route.notFound) return <ListingBody theme={theme} route={route} query={query} storePath={storePath} collection={null} />
  if (route.key === 'order-confirmation' && route.param) return <OrderProvider orderNumber={route.param}><Layout theme={theme} route={route} /></OrderProvider>
  return <Layout theme={theme} route={route} />
}

export function StorePage({ path, storeKey, query = new URLSearchParams() }: { path: string; storeKey: string; query?: URLSearchParams }) {
  const api = useStorefrontAPI()
  const { data: theme } = useSuspenseQuery(themeQuery(api))
  const { data: store } = useSuspenseQuery(storeQuery(api))
  const langs = storeLanguages(store)
  const { locale, path: pagePath } = splitLocale(path, langs)
  const route = documentFor(theme, pagePath)
  const storePath = storePathFor(storeKey, locale, langs)
  const qs = query.toString()
  const here = pagePath.replace(/^\/+/, '') + (qs ? `?${qs}` : '')
  const language = { storeKey, locale, offered: langs.offered, hrefFor: (l: Locale) => storePathFor(storeKey, l, langs)(here) }
  const i18n = useMemo(() => createI18n(locale), [locale])
  return (
    <I18nextProvider i18n={i18n}>
    <StoreLanguageProvider value={language}>
    <RenderProvider value={{ mode: 'live', storePath, contentLocale: locale === langs.defaultLocale ? undefined : locale }}>
      <StoreInfoProvider value={{ name: store?.name ?? '', settings: theme.settings, assets: theme.assets ?? {} }}>
        <CustomerProvider api={api} storeKey={storeKey}>
          <WishlistProvider api={api} storeKey={storeKey}>
            <CartProvider api={api} storeKey={storeKey}>
              <Routed theme={theme} route={route} query={query} storePath={storePath} />
            </CartProvider>
          </WishlistProvider>
        </CustomerProvider>
      </StoreInfoProvider>
    </RenderProvider>
    </StoreLanguageProvider>
    </I18nextProvider>
  )
}

/** The page language when there's no store to ask: the server picks it from
 *  the browser's Accept-Language and writes it to <html lang>, which the
 *  browser then reads back, so both render the same. */
function pageLocale(): Locale {
  return resolveLocale(typeof document === 'undefined' ? undefined : document.documentElement.lang)
}

export function StoreUnavailable({ locale = pageLocale() }: { locale?: Locale }) {
  const i18n = useMemo(() => createI18n(locale), [locale])
  const t = i18n.t
  return (
    <main className="cos-unavailable">
      <h1>{t('store.unavailableTitle')}</h1>
      <p>{t('store.unavailableText')}</p>
    </main>
  )
}
