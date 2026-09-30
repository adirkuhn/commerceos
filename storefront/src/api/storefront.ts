import type { DocumentKind, StorefrontTheme, ThemeDocument } from '../core/document'

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export interface StorefrontAPIConfig {
  /** e.g. https://api.example.com/api/v1 -- the SSR server uses an internal URL. */
  baseURL: string
  /** The channel's publishable storefront key. */
  storeKey: string
  /** Preview-session token: the draft is shown instead of the live theme. */
  previewToken?: string
  /** The shopper's language when it isn't the store's default: product
   *  and collection texts come back translated. */
  locale?: string
  fetch?: typeof fetch
}

export interface ProductSummary {
  id: string
  name: string
  slug: string
  category_id: string | null
  tags: string[]
  image_url: string | null
  price_minor_units?: number | null
  sale_price_minor_units?: number | null
  currency_code?: string | null
  /** Set when the product has exactly one SKU (a tile can add it). */
  sku_code?: string | null
  sku_count?: number
  in_stock?: boolean
}

export interface CategoryFacet {
  category_id: string
  category_name: string
  count: number
}

export interface ProductList {
  items: ProductSummary[]
  total: number
  category_facets?: CategoryFacet[]
}

export interface ProductQuery {
  categoryId?: string
  productIds?: string[]
  collectionId?: string
  search?: string
  page?: number
  perPage?: number
}

export interface ProductSKU {
  id: string
  sku_code: string
  price_minor_units: number
  sale_price_minor_units?: number | null
  currency_code: string
  stock_quantity: number
}

export interface ProductVariant {
  id: string
  name: string
  skus: ProductSKU[]
}

export interface ProductDetail {
  id: string
  name: string
  slug: string
  description: string
  category_id: string | null
  tags: string[]
  images: string[]
  variants: ProductVariant[]
}

export interface Review {
  id: string
  customer_name: string
  rating: number
  title: string
  body: string
  verified: boolean
  created_at: string
}

export interface ReviewList {
  reviews: Review[]
  summary: { average: number; count: number }
}

export interface Collection {
  id: string
  name: string
  slug: string
  description: string
  image_url?: string | null
  seo_title: string
  seo_description: string
}

export interface StoreInfo {
  name: string
  currency_code: string
  /** The storefront's default language (en | pt-BR | nl). */
  locale?: string
  /** Every language shoppers can switch to, the default first. */
  locales?: string[]
}

export interface CartItem {
  sku_code: string
  product_name: string
  quantity: number
  price_minor_units: number
  currency_code: string
}

export interface Account {
  email: string
  first_name: string
  last_name: string
}

export interface AccountOrder {
  order_number: string
  status: string
  fulfilled: boolean
  total_minor_units: number
  currency_code: string
  placed_at: string
}

export interface OrderStatus {
  order_number: string
  status: string
  fulfilled: boolean
  total_minor_units: number
  currency_code: string
}

export interface Wishlist {
  token: string
  product_ids: string[]
}

export interface Address {
  first_name: string
  last_name: string
  company?: string
  address1: string
  address2?: string
  city: string
  state?: string
  postcode: string
  country: string
  phone?: string
}

export interface ShippingOption {
  title: string
  amount_minor_units: number
  currency_code: string
}

export interface CheckoutResult {
  order_number: string
  total_minor_units: number
  currency_code: string
  payment_url?: string | null
}

export interface SitemapLink {
  slug: string
  updated_at: string
}

export interface SitemapData {
  products: SitemapLink[]
  collections: SitemapLink[]
}

export type NewsletterStatus = 'check_your_email' | 'subscribed' | 'unsubscribed'

export interface Cart {
  token: string
  items: CartItem[]
  subtotal_minor_units: number
  coupon_code?: string | null
  discount_minor_units: number
}

export interface Category {
  id: string
  parent_id: string | null
  name: string
  slug: string
  position: number
}

interface Envelope<T> {
  data: T | null
  error: { code?: string; message?: string } | null
}

/** Client for the public storefront API (/storefront/v1). */
export class StorefrontAPI {
  private readonly cfg: StorefrontAPIConfig

  constructor(cfg: StorefrontAPIConfig) {
    this.cfg = cfg
  }

  /** A copy asking for catalog texts in another language ("" = default). */
  inLocale(locale: string | undefined): StorefrontAPI {
    return locale === this.cfg.locale ? this : new StorefrontAPI({ ...this.cfg, locale: locale || undefined })
  }

  private localized(path: string): string {
    if (!this.cfg.locale) return path
    return `${path}${path.includes('?') ? '&' : '?'}locale=${encodeURIComponent(this.cfg.locale)}`
  }

  get previewing(): boolean {
    return Boolean(this.cfg.previewToken)
  }

  getTheme(): Promise<StorefrontTheme> {
    return this.get('/storefront/v1/theme')
  }

  getDocument(theme: StorefrontTheme, kind: DocumentKind, key: string): Promise<ThemeDocument> {
    const tail = `${kind}/${encodeURIComponent(key)}`
    return theme.preview
      ? this.get(`/storefront/v1/theme/preview/documents/${tail}`)
      : this.get(`/storefront/v1/theme/versions/${encodeURIComponent(theme.version_id)}/documents/${tail}`)
  }

  /** Products enrolled on the channel. `productIds` keeps exactly those. */
  listProducts(q: ProductQuery): Promise<ProductList> {
    const params = new URLSearchParams()
    if (q.categoryId) params.set('category_id', q.categoryId)
    if (q.productIds) params.set('product_ids', q.productIds.join(','))
    if (q.collectionId) params.set('collection_id', q.collectionId)
    if (q.search) params.set('search', q.search)
    if (q.page && q.page > 1) params.set('page', String(q.page))
    params.set('per_page', String(q.perPage ?? 12))
    if (this.cfg.locale) params.set('locale', this.cfg.locale)
    return this.get(`/storefront/v1/products?${params}`)
  }

  getProductBySlug(slug: string): Promise<ProductDetail> {
    return this.get(this.localized(`/storefront/v1/products/by-slug/${encodeURIComponent(slug)}`))
  }

  getRecommendations(productId: string): Promise<ProductSummary[]> {
    return this.get(this.localized(`/storefront/v1/products/${encodeURIComponent(productId)}/recommendations`))
  }

  listReviews(productId: string, perPage = 10): Promise<ReviewList> {
    return this.get(`/storefront/v1/products/${encodeURIComponent(productId)}/reviews?per_page=${perPage}`)
  }

  listCollections(): Promise<Collection[]> {
    return this.get(this.localized('/storefront/v1/collections?per_page=100'))
  }

  getCollection(slug: string): Promise<Collection> {
    return this.get(this.localized(`/storefront/v1/collections/by-slug/${encodeURIComponent(slug)}`))
  }

  getSitemap(): Promise<SitemapData> {
    return this.get('/storefront/v1/sitemap')
  }

  getStore(): Promise<StoreInfo> {
    return this.get('/storefront/v1/store')
  }

  createCart(): Promise<Cart> {
    return this.send('POST', '/storefront/v1/carts')
  }

  getCart(token: string): Promise<Cart> {
    return this.get(`/storefront/v1/carts/${encodeURIComponent(token)}`)
  }

  /** Replaces the cart's whole line set (the API has no incremental add). */
  setCartItems(token: string, items: { sku_code: string; quantity: number }[]): Promise<Cart> {
    return this.send('PUT', `/storefront/v1/carts/${encodeURIComponent(token)}/items`, { items })
  }

  applyCoupon(token: string, code: string): Promise<Cart> {
    return this.send('PUT', `/storefront/v1/carts/${encodeURIComponent(token)}/coupon`, { code })
  }

  /** Emails a sign-in link to the store's own login page (the server
   *  decides where it points, never the caller). */
  requestMagicLink(email: string): Promise<unknown> {
    return this.send('POST', '/storefront/v1/auth/magic-link', { email }, true)
  }

  verifySession(token: string): Promise<{ session_token: string; account: Account }> {
    return this.send('POST', '/storefront/v1/auth/session', { token })
  }

  getAccount(sessionToken: string): Promise<Account> {
    return this.get(`/storefront/v1/account/${encodeURIComponent(sessionToken)}`)
  }

  listAccountOrders(sessionToken: string): Promise<AccountOrder[]> {
    return this.get(`/storefront/v1/account/${encodeURIComponent(sessionToken)}/orders`)
  }

  logout(sessionToken: string): Promise<unknown> {
    return this.send('DELETE', `/storefront/v1/account/${encodeURIComponent(sessionToken)}`, undefined, true)
  }

  createWishlist(): Promise<Wishlist> {
    return this.send('POST', '/storefront/v1/wishlists')
  }

  getWishlist(token: string): Promise<Wishlist> {
    return this.get(`/storefront/v1/wishlists/${encodeURIComponent(token)}`)
  }

  toggleWishlistItem(token: string, productId: string): Promise<Wishlist> {
    return this.send('PUT', `/storefront/v1/wishlists/${encodeURIComponent(token)}/items`, { product_id: productId })
  }

  subscribeNewsletter(email: string, consent: boolean): Promise<{ status: NewsletterStatus }> {
    return this.send('POST', '/storefront/v1/newsletter/subscriptions', { email, consent })
  }

  confirmNewsletter(token: string): Promise<{ status: NewsletterStatus }> {
    return this.send('POST', '/storefront/v1/newsletter/confirmations', { token })
  }

  unsubscribeNewsletter(token: string): Promise<{ status: NewsletterStatus }> {
    return this.send('POST', '/storefront/v1/newsletter/unsubscriptions', { token })
  }

  shippingOptions(cartToken: string, address: Address): Promise<ShippingOption[]> {
    return this.send('POST', `/storefront/v1/carts/${encodeURIComponent(cartToken)}/shipping-options`, address)
  }

  checkout(cartToken: string, body: { email: string; billing: Address; shipping: Address; shipping_method_title?: string }): Promise<CheckoutResult> {
    return this.send('POST', `/storefront/v1/carts/${encodeURIComponent(cartToken)}/checkout`, body)
  }

  lookupOrder(orderNumber: string, email: string): Promise<OrderStatus> {
    const q = new URLSearchParams({ order_number: orderNumber, email })
    return this.get(`/storefront/v1/orders/lookup?${q}`)
  }

  listCategories(): Promise<Category[]> {
    return this.get('/storefront/v1/categories')
  }

  private get<T>(path: string): Promise<T> {
    return this.send('GET', path)
  }

  private async send<T>(method: string, path: string, body?: unknown, allowEmpty = false): Promise<T> {
    const headers = new Headers({ Accept: 'application/json', Authorization: `Bearer ${this.cfg.storeKey}` })
    if (this.cfg.previewToken) headers.set('X-Theme-Preview', this.cfg.previewToken)
    if (body !== undefined) headers.set('Content-Type', 'application/json')
    const doFetch = this.cfg.fetch ?? fetch
    const init: RequestInit = method === 'GET' ? { headers } : { method, headers, body: body === undefined ? undefined : JSON.stringify(body) }
    const res = await doFetch(`${this.cfg.baseURL}${path}`, init)
    let env: Envelope<T> | undefined
    try {
      env = (await res.json()) as Envelope<T>
    } catch {
      env = undefined
    }
    if (res.ok && allowEmpty && (!env || env.data === null || env.data === undefined)) return undefined as T
    if (!res.ok || !env || env.data === null) {
      throw new ApiError(res.status, env?.error?.code ?? 'HTTP_ERROR', env?.error?.message ?? `request failed (${res.status})`)
    }
    return env.data
  }
}
