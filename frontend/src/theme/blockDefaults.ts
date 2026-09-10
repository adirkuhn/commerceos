import type { Block, BlockType, Rect } from './types'

function newBlockId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `block-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const DEFAULT_PROPS: Record<BlockType, Record<string, unknown>> = {
  hero: { heading: 'Welcome', subheading: '', imageUrl: '', ctaText: '', ctaUrl: '' },
  product_grid: {},
  featured_products: {
    title: 'Featured products', subtitle: '', categoryId: '', productIds: [], excludeProductIds: [],
    limit: 8, itemsPerRowDesktop: 4, itemsPerRowMobile: 2,
  },
  rich_text: { content: 'Edit this text…' },
  custom_html: { html: '' },
  product_details: {},
  cart_summary: {},
  checkout_form: {},
  order_status: {},
  wishlist: {},
  login_form: {},
  account: {},
}

// Decorative blocks start as a small box a merchant places by hand. The
// rest are real page sections (a product grid, a whole checkout form) --
// they start big enough to look like one, though every dimension stays
// freely resizable afterward.
const DEFAULT_RECT: Record<BlockType, Rect> = {
  hero: { x: 20, y: 20, width: 320, height: 160 },
  rich_text: { x: 20, y: 20, width: 320, height: 160 },
  custom_html: { x: 20, y: 20, width: 320, height: 160 },
  product_grid: { x: 0, y: 20, width: 1440, height: 600 },
  featured_products: { x: 0, y: 20, width: 1440, height: 420 },
  product_details: { x: 0, y: 20, width: 1440, height: 600 },
  cart_summary: { x: 0, y: 20, width: 1440, height: 600 },
  checkout_form: { x: 0, y: 20, width: 1440, height: 600 },
  order_status: { x: 0, y: 20, width: 1440, height: 400 },
  wishlist: { x: 0, y: 20, width: 1440, height: 600 },
  login_form: { x: 0, y: 20, width: 480, height: 400 },
  account: { x: 0, y: 20, width: 1440, height: 600 },
}

export function createBlock(type: BlockType): Block {
  return {
    id: newBlockId(),
    type,
    layout: { desktop: { ...DEFAULT_RECT[type] }, mobile: null },
    props: { ...DEFAULT_PROPS[type] },
  }
}
