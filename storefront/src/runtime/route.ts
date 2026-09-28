import type { DocumentKind } from '../core/document'

// Maps a storefront path (below /store/{channelKey}) to the theme
// document that renders it.

export interface ResolvedRoute {
  kind: DocumentKind
  key: string
  /** The dynamic segment (product slug, collection slug, order number). */
  param?: string
  notFound: boolean
  /** A page the storefront itself provides (inside the theme's header
   *  and footer) rather than a theme document. */
  builtin?: BuiltinPage
}

export type BuiltinPage = 'newsletter-confirm' | 'newsletter-unsubscribe'

const BUILTIN_PAGES: Record<string, BuiltinPage> = {
  'newsletter/confirm': 'newsletter-confirm',
  'newsletter/unsubscribe': 'newsletter-unsubscribe',
}

const PAGE_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
const SEGMENT = /^[^/]+$/

const STATIC_TEMPLATES: Record<string, string> = {
  '': 'home', search: 'search', cart: 'cart', checkout: 'checkout', login: 'login', account: 'account', wishlist: 'wishlist',
}

const DYNAMIC_TEMPLATES: Record<string, string> = {
  products: 'product', collections: 'collection', orders: 'order-confirmation',
}

const NOT_FOUND: ResolvedRoute = { kind: 'template', key: 'not-found', notFound: true }

export function resolveRoute(path: string): ResolvedRoute {
  const parts = path.split('/').filter(Boolean)
  if (parts.length <= 1 && STATIC_TEMPLATES[parts[0] ?? ''] !== undefined) {
    return { kind: 'template', key: STATIC_TEMPLATES[parts[0] ?? ''], notFound: false }
  }
  if (parts.length === 2) {
    const builtin = BUILTIN_PAGES[parts.join('/')]
    if (builtin) return { kind: 'template', key: builtin, builtin, notFound: false }
    const [section, param] = parts
    if (section === 'pages') return PAGE_SLUG.test(param) ? { kind: 'page', key: param, notFound: false } : NOT_FOUND
    const template = DYNAMIC_TEMPLATES[section]
    if (template && SEGMENT.test(param)) return { kind: 'template', key: template, param, notFound: false }
  }
  return NOT_FOUND
}
