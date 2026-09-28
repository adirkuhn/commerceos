import type { StorefrontTheme, ThemeDocument } from '../core/document'
import { fontStylesheetURL, settingsToCSS } from '../core/settings'
import type { PageData } from './page/routeData'
import { breadcrumbJsonLd, indexable, jsonLdScript, organizationJsonLd, productJsonLd } from './seo'

export interface HeadInput {
  theme: StorefrontTheme
  /** The route's document (absent for built-in pages). */
  doc?: ThemeDocument
  page: PageData
  preview: boolean
  storeName: string
  /** Absolute URL of this page; absent without a configured public origin. */
  canonical?: string
  /** Absolute URL of the store's home page. */
  homeURL?: string
  rating?: { average: number; count: number }
}

const escapeHTML = (s: string) => s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)

const meta = (attr: 'name' | 'property', key: string, value: string | undefined) =>
  value ? `<meta ${attr}="${key}" content="${escapeHTML(value)}">` : ''

function structuredData({ theme, page, storeName, canonical, homeURL, rating }: HeadInput): string[] {
  const { route, product, collection } = page
  if (product) {
    const crumbs = homeURL && canonical ? [jsonLdScript(breadcrumbJsonLd([{ name: storeName || 'Home', url: homeURL }, { name: product.name, url: canonical }]))] : []
    return [jsonLdScript(productJsonLd(product, canonical, rating)), ...crumbs]
  }
  if (collection && homeURL && canonical) {
    return [jsonLdScript(breadcrumbJsonLd([{ name: storeName || 'Home', url: homeURL }, { name: collection.name, url: canonical }]))]
  }
  if (route.key === 'home' && homeURL && storeName) {
    const logoID = theme.settings.brand.logo_asset_id
    return [jsonLdScript(organizationJsonLd(storeName, homeURL, logoID ? theme.assets?.[logoID]?.url : undefined))]
  }
  return []
}

/** Everything a server-rendered page puts in <head>. */
export function buildHead(input: HeadInput): string {
  const { theme, doc, page, preview, storeName, canonical } = input
  const title = page.seoTitle || doc?.seo?.title || doc?.title || storeName || 'Store'
  const description = page.seoDescription || doc?.seo?.description
  const index = !preview && indexable(page.route)
  const image = page.product?.images[0] ?? page.collection?.image_url ?? undefined

  const parts = [`<title>${escapeHTML(title)}</title>`, meta('name', 'description', description)]
  if (!index) parts.push('<meta name="robots" content="noindex">')
  // Built-in pages carry one-time tokens in their URL: keep them out of
  // Referer headers too.
  if (page.route.builtin) parts.push('<meta name="referrer" content="no-referrer">')
  if (index) {
    if (canonical) parts.push(`<link rel="canonical" href="${escapeHTML(canonical)}">`)
    parts.push(
      meta('property', 'og:type', page.product ? 'product' : 'website'),
      meta('property', 'og:title', title),
      meta('property', 'og:description', description),
      meta('property', 'og:url', canonical),
      meta('property', 'og:image', image),
      meta('property', 'og:site_name', storeName),
      ...structuredData(input),
    )
  }
  const favicon = theme.settings.brand.favicon_asset_id && theme.assets?.[theme.settings.brand.favicon_asset_id]
  if (favicon) parts.push(`<link rel="icon" href="${escapeHTML(favicon.url)}">`)
  const fonts = fontStylesheetURL(theme.settings)
  if (fonts) parts.push('<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>', `<link rel="stylesheet" href="${escapeHTML(fonts)}">`)
  parts.push(`<style>${settingsToCSS(theme.settings)}</style>`)
  return parts.filter(Boolean).join('')
}
