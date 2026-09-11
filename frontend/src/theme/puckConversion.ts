import type { Block, BlockType } from './types'

// PUCK_COMPONENT_NAME maps this system's own BlockType (freeform) onto
// storefront-frontend's registered Puck component names (see
// storefront-frontend/src/puck/config.tsx's buildPuckConfig) -- the two
// naming conventions differ (snake_case vs PascalCase) even though every
// block type has a 1:1 counterpart; there is no BlockType Puck doesn't
// support.
const PUCK_COMPONENT_NAME: Record<BlockType, string> = {
  hero: 'Hero',
  product_grid: 'ProductGrid',
  featured_products: 'FeaturedProducts',
  rich_text: 'RichText',
  custom_html: 'CustomHTML',
  nav_menu: 'NavMenu',
  product_details: 'ProductDetails',
  cart_summary: 'CartSummary',
  checkout_form: 'CheckoutForm',
  order_status: 'OrderStatus',
  wishlist: 'Wishlist',
  login_form: 'LoginForm',
  account: 'Account',
  nav_actions: 'NavActions',
}

function toStringList(value: unknown): string {
  return Array.isArray(value) ? value.filter((v): v is string => typeof v === 'string').join(', ') : ''
}

// puckPropsFor re-shapes a block's own props into what that Puck
// component's fields expect (see config.tsx) -- everything is a 1:1
// passthrough except featured_products' id-list fields, which Puck's own
// FeaturedProducts.fields store as a comma-separated string (parsed back
// out via config.tsx's own parseIdList) rather than this system's array.
function puckPropsFor(block: Block): Record<string, unknown> {
  if (block.type === 'featured_products') {
    return {
      ...block.props,
      productIds: toStringList(block.props.productIds),
      excludeProductIds: toStringList(block.props.excludeProductIds),
    }
  }
  return { ...block.props }
}

// blocksToPuckData converts this system's own freeform Block[] (absolute
// x/y canvas positions + typed props) into Puck's own Data shape (an
// ordered content[] with no position info -- Puck lays components out in
// document order within a single vertical flow, not a free canvas) --
// used both to seed a brand-new internal channel straight into Puck (see
// SalesChannels.tsx's seedStarterTheme) and to give an already-existing
// freeform page a real starting point the first time it's opened/migrated
// into Puck, instead of Puck's own empty-page default (see
// storefront-frontend EditPage.tsx's EMPTY_PUCK_DATA fallback). Position
// is deliberately dropped, not approximated -- Puck has no equivalent
// concept, and document order (already how build*Blocks lists every
// block, top-to-bottom/reading order) is the correct substitute.
export function blocksToPuckData(blocks: Block[]): Record<string, unknown> {
  return {
    content: blocks.map(block => ({
      type: PUCK_COMPONENT_NAME[block.type],
      props: { id: block.id, ...puckPropsFor(block) },
    })),
    root: {},
  }
}
