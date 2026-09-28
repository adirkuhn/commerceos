import type { ProductDetail } from '../api/storefront'
import { decimalAmount } from '../core/money'
import { htmlToParagraphs } from '../core/plainText'
import type { ResolvedRoute } from './route'

// Search-engine facing parts of a page: canonical URL, whether it may be
// indexed, and schema.org structured data. Pure, so SSR and tests agree.

const SCHEMA = 'https://schema.org'

/** Pages that are per-shopper or utility screens, never search results. */
const PRIVATE_TEMPLATES = new Set(['cart', 'checkout', 'account', 'login', 'wishlist', 'order-confirmation', 'search', 'not-found'])

export function indexable(route: ResolvedRoute): boolean {
  if (route.builtin || route.notFound) return false
  return !(route.kind === 'template' && PRIVATE_TEMPLATES.has(route.key))
}

/** The page's public address. Only a listing's page number survives from
 *  the query (filters and tracking params would split rankings). */
export function canonicalURL(origin: string | undefined, storeKey: string, path: string, query: URLSearchParams, route: ResolvedRoute): string | undefined {
  if (!origin) return undefined
  const base = `${origin.replace(/\/+$/, '')}/store/${encodeURIComponent(storeKey)}/${path.replace(/^\/+/, '')}`
  const page = Math.floor(Number(query.get('page')))
  return route.key === 'collection' && page > 1 ? `${base}?page=${page}` : base
}

export function plainDescription(html: string, max = 160): string {
  return htmlToParagraphs(html).join(' ').replace(/\s+/g, ' ').trim().slice(0, max)
}

type JsonLd = Record<string, unknown>

export function productJsonLd(product: ProductDetail, url: string | undefined, rating?: { average: number; count: number }): JsonLd {
  const skus = product.variants.flatMap(v => v.skus)
  const charged = (s: (typeof skus)[number]) => s.sale_price_minor_units ?? s.price_minor_units
  const availability = `${SCHEMA}/${skus.some(s => s.stock_quantity > 0) ? 'InStock' : 'OutOfStock'}`
  const currency = skus[0]?.currency_code
  let offers: JsonLd | undefined
  if (skus.length === 1 && currency) {
    offers = { '@type': 'Offer', price: decimalAmount(charged(skus[0]), currency), priceCurrency: currency, availability, ...(url ? { url } : {}) }
  } else if (skus.length > 1 && currency) {
    const prices = skus.map(charged)
    offers = {
      '@type': 'AggregateOffer', lowPrice: decimalAmount(Math.min(...prices), currency), highPrice: decimalAmount(Math.max(...prices), currency),
      offerCount: skus.length, priceCurrency: currency, availability,
    }
  }
  const description = plainDescription(product.description, 5000)
  return {
    '@context': SCHEMA, '@type': 'Product', name: product.name,
    ...(description ? { description } : {}),
    ...(product.images.length ? { image: product.images } : {}),
    ...(skus.length === 1 ? { sku: skus[0].sku_code } : {}),
    ...(url ? { url } : {}),
    ...(offers ? { offers } : {}),
    ...(rating && rating.count > 0 ? { aggregateRating: { '@type': 'AggregateRating', ratingValue: rating.average, reviewCount: rating.count } } : {}),
  }
}

export function breadcrumbJsonLd(items: { name: string; url: string }[]): JsonLd {
  return {
    '@context': SCHEMA, '@type': 'BreadcrumbList',
    itemListElement: items.map((it, i) => ({ '@type': 'ListItem', position: i + 1, name: it.name, item: it.url })),
  }
}

export function organizationJsonLd(name: string, url: string, logo?: string): JsonLd {
  return { '@context': SCHEMA, '@type': 'Organization', name, url, ...(logo ? { logo } : {}) }
}

/** `<` is escaped so catalog text can't end the script element. */
export function jsonLdScript(data: JsonLd): string {
  return `<script type="application/ld+json">${JSON.stringify(data).replace(/</g, '\\u003c')}</script>`
}
