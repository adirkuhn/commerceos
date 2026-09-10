export interface APIResponse<T> {
  data: T | null
  error: { code: string; message: string } | null
}

export type PageSlug =
  | 'home'
  | 'products'
  | 'product'
  | 'cart'
  | 'checkout'
  | 'order-confirmation'
  | 'order-lookup'
  | 'wishlist'
  | 'login'
  | 'account'

export type BlockType =
  | 'hero'
  | 'product_grid'
  | 'featured_products'
  | 'rich_text'
  | 'custom_html'
  | 'product_details'
  | 'cart_summary'
  | 'checkout_form'
  | 'order_status'
  | 'wishlist'
  | 'login_form'
  | 'account'

export interface Rect {
  x: number
  y: number
  width: number
  height: number
}

export interface BlockLayout {
  desktop: Rect
  mobile: Rect | null
}

export interface Block {
  id: string
  type: BlockType
  layout: BlockLayout
  props: Record<string, unknown>
}

// EditorMode selects which of ThemePage's two documents CanvasPage
// actually renders from -- 'freeform' (blocks, the absolutely-positioned
// canvas below) or 'puck' (puck_data, authored by the embedded Puck
// editor at /edit/:slug -- see puck/config.tsx). Mirrors backend
// domain/theme.EditorMode.
export type EditorMode = 'freeform' | 'puck'

export interface ThemePage {
  slug: PageSlug
  canvas_width: number
  blocks: Block[]
  custom_css: string
  editor_mode: EditorMode
  // Opaque -- see domain/theme.ThemePage.PuckData's doc comment. Only
  // meaningful when editor_mode is 'puck'.
  puck_data?: Record<string, unknown> | null
  // Only present on the embedded-editor draft response (see
  // api/themeDraft.ts) -- the public embed (api/theme.ts) never sends
  // it, a shopper's own key already scopes that request. EditPage uses
  // it to scope the editor's own live blocks (see
  // api/client.ts.setStorefrontKey) to the right tenant/channel on this
  // multi-tenant deployment.
  sales_channel_id?: string
}

// Mirrors backend domain/theme.PageSlug.RequiredBlockType: these pages
// can't work without their functional block, so a merchant is free to
// move/resize it (or never touch the page at all) but never to delete
// real functionality by leaving it out -- see CanvasPage's fallback.
export const REQUIRED_BLOCK_TYPE_BY_SLUG: Partial<Record<PageSlug, BlockType>> = {
  product: 'product_details',
  cart: 'cart_summary',
  checkout: 'checkout_form',
  'order-confirmation': 'order_status',
  'order-lookup': 'order_status',
  wishlist: 'wishlist',
  login: 'login_form',
  account: 'account',
}

export interface Wishlist {
  token: string
  product_ids: string[]
}

export interface Account {
  email: string
  first_name: string
  last_name: string
}

export interface Session {
  session_token: string
  account: Account
}

export interface AccountOrder {
  order_number: string
  status: string
  fulfilled: boolean
  total_minor_units: number
  currency_code: string
  placed_at: string
}

export interface CartItem {
  sku_code: string
  product_name: string
  quantity: number
  price_minor_units: number
  currency_code: string
}

export interface Cart {
  token: string
  items: CartItem[]
  subtotal_minor_units: number
  coupon_code?: string | null
  discount_minor_units: number
}

export interface UpdateCartItemLine {
  sku_code: string
  quantity: number
}

export interface CheckoutAddress {
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

export interface CheckoutRequest {
  email: string
  billing: CheckoutAddress
  shipping: CheckoutAddress
  shipping_method_title?: string
  payment_intent_id?: string
}

export interface Checkout {
  order_number: string
  total_minor_units: number
  currency_code: string
  payment_url?: string
}

export interface CreatePaymentIntentRequest {
  shipping: CheckoutAddress
  shipping_method_title?: string
}

export interface PaymentIntent {
  client_secret: string
  publishable_key: string
  amount_minor_units: number
  currency_code: string
}

export interface ShippingOption {
  title: string
  amount_minor_units: number
  currency_code: string
}

export interface StorefrontProductSummary {
  id: string
  slug: string
  name: string
  category_id: string | null
  tags: string[]
  image_url: string | null
  price_minor_units?: number
  sale_price_minor_units?: number
  currency_code?: string
}

export interface StorefrontCategoryFacet {
  category_id: string
  category_name: string
  count: number
}

export interface StorefrontProductListResult {
  items: StorefrontProductSummary[]
  total: number
  category_facets: StorefrontCategoryFacet[]
}

export interface StorefrontSKU {
  id: string
  sku_code: string
  price_minor_units: number
  sale_price_minor_units?: number
  currency_code: string
  stock_quantity: number
}

export interface StorefrontVariant {
  id: string
  name: string
  skus: StorefrontSKU[]
}

export interface StorefrontProduct {
  id: string
  slug: string
  name: string
  description: string
  category_id: string | null
  tags: string[]
  images: string[]
  variants: StorefrontVariant[]
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

export interface ReviewSummary {
  average: number
  count: number
}

export interface ReviewList {
  reviews: Review[]
  summary: ReviewSummary
}

export interface SubmitReviewInput {
  customer_name: string
  customer_email: string
  rating: number
  title?: string
  body: string
}

export interface StorefrontCategory {
  id: string
  parent_id: string | null
  name: string
  slug: string
  position: number
}

export interface StorefrontOrderStatus {
  order_number: string
  status: string
  fulfilled: boolean
  total_minor_units: number
  currency_code: string
}
