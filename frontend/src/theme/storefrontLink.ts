import type { PageSlug } from './types'

// storefrontOrigin is where storefront-frontend is actually reachable.
// VITE_STOREFRONT_URL overrides this once it's deployed on a dedicated
// domain of its own; until then it defaults to VITE_API_BASE_URL's own
// origin, since that's where it's path-mounted by default (see
// docker-compose.yml's storefront service) -- a real, working link with
// no extra configuration step. "" only if that default can't be computed
// (VITE_API_BASE_URL isn't a valid URL).
export function storefrontOrigin(): string {
  const explicit = import.meta.env.VITE_STOREFRONT_URL
  if (explicit) return explicit
  try {
    return new URL(import.meta.env.VITE_API_BASE_URL).origin
  } catch {
    return ''
  }
}

// storefrontEnterPath is every internal channel's own storefront entry
// point -- the channel's own id doubles as its storefront bearer key (see
// backend application/apikeys.ResolveAPIKeyUseCase's fallback). next, when
// given, is where storefront-frontend's StoreEntry sends the browser after
// remembering the key (defaults to "/", the home page) -- used to preview
// a specific page instead of always landing on home.
export function storefrontEnterPath(channelId: string, next?: string): string {
  const path = `/store/enter/${channelId}`
  return next ? `${path}?next=${encodeURIComponent(next)}` : path
}

// storefrontVisitURL is the full clickable "Visit store" link.
export function storefrontVisitURL(channelId: string): string {
  const origin = storefrontOrigin()
  return origin ? `${origin}${storefrontEnterPath(channelId)}` : ''
}

// SLUG_PREVIEW_PATH maps a theme page slug to the real storefront route
// that renders it, for the "Preview" button in the theme editor. Omits
// slugs with no fixed route to preview: 'product' needs a real product id
// (falls back to the products list, the closest fixed-path equivalent),
// 'order-confirmation' needs a real just-placed order number and can't be
// previewed generically at all.
export const SLUG_PREVIEW_PATH: Partial<Record<PageSlug, string>> = {
  home: '/',
  products: '/products',
  product: '/products',
  cart: '/cart',
  checkout: '/checkout',
  'order-lookup': '/order-lookup',
  wishlist: '/wishlist',
  login: '/login',
  account: '/account',
}

// storefrontPreviewURL is the full URL the theme editor's "Preview" button
// opens -- the real, saved, live-rendered storefront page (real styling,
// real product data, real custom HTML), not the editor's own rough
// canvas approximation. "" when this slug has no previewable route (see
// SLUG_PREVIEW_PATH) or the origin can't be computed.
export function storefrontPreviewURL(channelId: string, slug: PageSlug): string {
  const path = SLUG_PREVIEW_PATH[slug]
  if (!path) return ''
  const origin = storefrontOrigin()
  return origin ? `${origin}${storefrontEnterPath(channelId, path)}` : ''
}
