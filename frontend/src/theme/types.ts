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

export const PAGE_SLUGS: PageSlug[] = [
  'home', 'products', 'product', 'cart', 'checkout', 'order-confirmation', 'order-lookup',
]

export const PAGE_SLUG_LABELS: Record<PageSlug, string> = {
  home: 'Home',
  products: 'Products',
  product: 'Product',
  cart: 'Cart',
  checkout: 'Checkout',
  'order-confirmation': 'Order confirmation',
  'order-lookup': 'Order lookup',
}

export type BlockType =
  | 'hero'
  | 'product_grid'
  | 'rich_text'
  | 'custom_html'
  | 'product_details'
  | 'cart_summary'
  | 'checkout_form'
  | 'order_status'

// Every block type is addable from the canvas -- including the
// functional ones, which otherwise only ever appear at their
// storefront-synthesized default position (see storefront-frontend
// CanvasPage.withRequiredBlock). Adding one here lets a merchant
// reposition/resize the real thing; it never has editable props (its
// content is live cart/catalog/order data, not merchant text) so
// BlockPropsPanel shows no fields for it beyond delete.
export const EDITABLE_BLOCK_TYPES: BlockType[] = [
  'hero', 'rich_text', 'custom_html',
  'product_grid', 'product_details', 'cart_summary', 'checkout_form', 'order_status',
]

export const BLOCK_TYPE_LABELS: Record<BlockType, string> = {
  hero: 'Hero',
  product_grid: 'Product grid',
  rich_text: 'Rich text',
  custom_html: 'Custom HTML',
  product_details: 'Product details',
  cart_summary: 'Cart summary',
  checkout_form: 'Checkout form',
  order_status: 'Order status',
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

export interface ThemePage {
  sales_channel_id: string
  slug: PageSlug
  canvas_width: number
  blocks: Block[]
}

export interface SaveThemePagePayload {
  canvas_width: number
  blocks: Block[]
}
