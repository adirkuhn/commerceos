import { ProductGridBlock } from './blocks/ProductGridBlock'
import { FeaturedProductsBlock } from './blocks/FeaturedProductsBlock'
import { ProductDetailsBlock } from './blocks/ProductDetailsBlock'
import { CartSummaryBlock } from './blocks/CartSummaryBlock'
import { CheckoutFormBlock } from './blocks/CheckoutFormBlock'
import { OrderStatusBlock } from './blocks/OrderStatusBlock'
import { WishlistBlock } from './blocks/WishlistBlock'
import { LoginFormBlock } from './blocks/LoginFormBlock'
import { AccountBlock } from './blocks/AccountBlock'
import { NavMenuBlock, type NavMenuLink } from './blocks/NavMenuBlock'
import { NavActionsBlock } from './blocks/NavActionsBlock'
import type { Block, PageSlug } from '../types'

function stringProp(props: Record<string, unknown>, key: string): string {
  const v = props[key]
  return typeof v === 'string' ? v : ''
}

function numberProp(props: Record<string, unknown>, key: string, fallback: number): number {
  const v = props[key]
  return typeof v === 'number' ? v : fallback
}

function stringArrayProp(props: Record<string, unknown>, key: string): string[] | undefined {
  const v = props[key]
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : undefined
}

function navLinksProp(props: Record<string, unknown>, key: string): NavMenuLink[] {
  const v = props[key]
  if (!Array.isArray(v)) return []
  return v.filter((x): x is NavMenuLink =>
    typeof x === 'object' && x !== null && typeof (x as NavMenuLink).label === 'string' && typeof (x as NavMenuLink).url === 'string')
}

interface Props {
  block: Block
  pageSlug: PageSlug
}

// Renders one block's real content. hero/rich_text/custom_html render
// merchant-authored props directly; the functional types (see backend
// domain/theme.BlockType.IsFunctional) delegate to a real, wired-up
// component under ./blocks -- they ignore props entirely, since their
// content is live cart/catalog/order data, not merchant-authored text.
export function BlockRenderer({ block, pageSlug }: Props) {
  switch (block.type) {
    case 'hero': {
      const heading = stringProp(block.props, 'heading')
      const subheading = stringProp(block.props, 'subheading')
      const imageUrl = stringProp(block.props, 'imageUrl')
      const ctaText = stringProp(block.props, 'ctaText')
      const ctaUrl = stringProp(block.props, 'ctaUrl')
      return (
        <div
          className="flex h-full w-full flex-col justify-center gap-2 bg-brand-50 px-8"
          style={imageUrl ? { backgroundImage: `url(${imageUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : undefined}
        >
          {heading && <h2 className="text-3xl font-semibold text-brand-700">{heading}</h2>}
          {subheading && <p className="text-brand-500">{subheading}</p>}
          {ctaText && ctaUrl && (
            <a href={ctaUrl} className="mt-2 inline-block w-fit rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
              {ctaText}
            </a>
          )}
        </div>
      )
    }
    case 'featured_products':
      return (
        <FeaturedProductsBlock
          title={stringProp(block.props, 'title')}
          subtitle={stringProp(block.props, 'subtitle') || undefined}
          categoryId={stringProp(block.props, 'categoryId') || undefined}
          productIds={stringArrayProp(block.props, 'productIds')}
          excludeProductIds={stringArrayProp(block.props, 'excludeProductIds')}
          limit={numberProp(block.props, 'limit', 8)}
          itemsPerRowDesktop={numberProp(block.props, 'itemsPerRowDesktop', 4)}
          itemsPerRowMobile={numberProp(block.props, 'itemsPerRowMobile', 2)}
        />
      )
    case 'rich_text': {
      const content = stringProp(block.props, 'content')
      return (
        <div className="h-full w-full overflow-auto whitespace-pre-wrap px-2 py-1 text-brand-700">
          {content}
        </div>
      )
    }
    case 'custom_html': {
      // Admin-authored (see infrastructure/handlers/theme -- JWT-gated,
      // never shopper input), same trust boundary as any other merchant
      // theme-customization surface.
      const html = stringProp(block.props, 'html')
      return <div className="h-full w-full overflow-auto" dangerouslySetInnerHTML={{ __html: html }} />
    }
    case 'product_grid':
      return <ProductGridBlock />
    case 'product_details':
      return <ProductDetailsBlock />
    case 'cart_summary':
      return <CartSummaryBlock />
    case 'checkout_form':
      return <CheckoutFormBlock />
    case 'order_status':
      return <OrderStatusBlock pageSlug={pageSlug === 'order-lookup' ? 'order-lookup' : 'order-confirmation'} />
    case 'wishlist':
      return <WishlistBlock />
    case 'login_form':
      return <LoginFormBlock />
    case 'account':
      return <AccountBlock />
    case 'nav_menu':
      return <NavMenuBlock links={navLinksProp(block.props, 'links')} />
    case 'nav_actions':
      return <NavActionsBlock />
    default:
      return (
        <div className="flex h-full w-full items-center justify-center border border-dashed border-gray-300 bg-gray-50">
          <p className="text-sm text-gray-400">{block.type} — unsupported block type</p>
        </div>
      )
  }
}
