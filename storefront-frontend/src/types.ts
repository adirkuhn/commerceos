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

export type BlockType =
  | 'hero'
  | 'product_grid'
  | 'rich_text'
  | 'custom_html'
  | 'product_details'
  | 'cart_summary'
  | 'checkout_form'
  | 'order_status'

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
  slug: PageSlug
  canvas_width: number
  blocks: Block[]
}
