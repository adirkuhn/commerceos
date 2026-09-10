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
  // header/footer are site-wide "pseudo-pages", edited exactly like any
  // real page -- see backend domain/theme.PageSlugHeader's own doc
  // comment. storefront-frontend's Layout renders them around every real
  // page's own content, not a shopper route of their own.
  | 'header'
  | 'footer'

export const PAGE_SLUGS: PageSlug[] = [
  'home', 'products', 'product', 'cart', 'checkout', 'order-confirmation', 'order-lookup', 'wishlist', 'login', 'account',
  'header', 'footer',
]

export const PAGE_SLUG_LABELS: Record<PageSlug, string> = {
  home: 'Home',
  products: 'Products',
  product: 'Product',
  cart: 'Cart',
  checkout: 'Checkout',
  'order-confirmation': 'Order confirmation',
  'order-lookup': 'Order lookup',
  wishlist: 'Wishlist',
  login: 'Login',
  account: 'Account',
  header: 'Header',
  footer: 'Footer',
}

export type BlockType =
  | 'hero'
  | 'product_grid'
  | 'featured_products'
  | 'rich_text'
  | 'custom_html'
  // nav_menu is a merchant-ordered list of label/url links -- reusable on
  // both the header page (category browse links) and the footer page
  // (privacy/terms/returns links).
  | 'nav_menu'
  | 'product_details'
  | 'cart_summary'
  | 'checkout_form'
  | 'order_status'
  | 'wishlist'
  | 'login_form'
  | 'account'
  // nav_actions is the header page's one required functional block -- the
  // real, live Cart/Wishlist/Account links. See backend
  // domain/theme.BlockTypeNavActions' own doc comment.
  | 'nav_actions'

// Every block type is addable from the canvas -- including the
// functional ones, which otherwise only ever appear at their
// storefront-synthesized default position (see storefront-frontend
// CanvasPage.withRequiredBlock). Adding one here lets a merchant
// reposition/resize the real thing; it never has editable props (its
// content is live cart/catalog/order data, not merchant text) so
// BlockPropsPanel shows no fields for it beyond delete.
export const EDITABLE_BLOCK_TYPES: BlockType[] = [
  'hero', 'rich_text', 'custom_html', 'nav_menu',
  'product_grid', 'featured_products', 'product_details', 'cart_summary', 'checkout_form', 'order_status', 'wishlist', 'login_form', 'account',
  'nav_actions',
]

export const BLOCK_TYPE_LABELS: Record<BlockType, string> = {
  hero: 'Hero',
  product_grid: 'Product grid',
  featured_products: 'Featured products',
  rich_text: 'Rich text',
  custom_html: 'Custom HTML',
  nav_menu: 'Nav menu',
  product_details: 'Product details',
  cart_summary: 'Cart summary',
  checkout_form: 'Checkout form',
  order_status: 'Order status',
  wishlist: 'Wishlist',
  login_form: 'Login form',
  account: 'Account',
  nav_actions: 'Nav actions (cart/wishlist/account)',
}

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

// EditorMode selects which document storefront-frontend renders a page
// from -- 'freeform' (this editor's own absolutely-positioned canvas,
// below) or 'puck' (authored by storefront-frontend's own embedded Puck
// editor at /edit/:slug -- see ThemeEditorPage's "Edit visually"
// launcher). Both documents are kept on every row regardless of which is
// active -- mirrors backend domain/theme.EditorMode.
export type EditorMode = 'freeform' | 'puck'

export interface ThemePage {
  sales_channel_id: string
  slug: PageSlug
  canvas_width: number
  blocks: Block[]
  custom_css: string
  editor_mode: EditorMode
  // Opaque -- this editor never reads/writes its contents, only round-trips
  // it unchanged on every freeform save so switching modes never loses the
  // other one's content. See domain/theme.ThemePage.PuckData's doc comment.
  puck_data?: Record<string, unknown> | null
}

export interface SaveThemePagePayload {
  canvas_width: number
  blocks: Block[]
  custom_css: string
  editor_mode: EditorMode
  puck_data?: Record<string, unknown> | null
}

export interface EditSession {
  token: string
  expires_at: string
}
