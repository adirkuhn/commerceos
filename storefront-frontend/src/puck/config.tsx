import type { Config } from '@puckeditor/core'
import { ProductGridBlock } from '../components/blocks/ProductGridBlock'
import { FeaturedProductsBlock } from '../components/blocks/FeaturedProductsBlock'
import { ProductDetailsBlock } from '../components/blocks/ProductDetailsBlock'
import { CartSummaryBlock } from '../components/blocks/CartSummaryBlock'
import { CheckoutFormBlock } from '../components/blocks/CheckoutFormBlock'
import { OrderStatusBlock } from '../components/blocks/OrderStatusBlock'
import { WishlistBlock } from '../components/blocks/WishlistBlock'
import { LoginFormBlock } from '../components/blocks/LoginFormBlock'
import { AccountBlock } from '../components/blocks/AccountBlock'
import { NavMenuBlock, type NavMenuLink } from '../components/blocks/NavMenuBlock'
import { NavActionsBlock } from '../components/blocks/NavActionsBlock'
import type { PageSlug } from '../types'

function parseIdList(value: unknown): string[] | undefined {
  if (typeof value !== 'string' || value.trim() === '') return undefined
  return value.split(',').map(id => id.trim()).filter(id => id !== '')
}

// buildPuckConfig registers every storefront block as a Puck component --
// unlike the admin app's own freeform-canvas editor (BlockPreview, a rough
// approximation), the functional ones here ARE the real, live, wired-up
// component (real cart/checkout/product data), not a placeholder -- that's
// the whole point of embedding Puck inside this app instead of the admin
// one (see frontend/pages/ThemeEditorPage.tsx's "Edit visually" launcher).
//
// A functional block still needs *real* surrounding context to render
// meaningfully (ProductDetailsBlock needs a real :id from the route,
// CheckoutFormBlock needs a real cart) -- editing a page whose slug
// doesn't naturally provide that (e.g. placing ProductDetailsBlock while
// editing 'home') is a known rough edge: the component renders whatever
// its own empty/not-found state already is, same as a shopper hitting
// that state for real, not a crash.
//
// slug is baked in per editor/render instance (not stored per-component)
// because OrderStatusBlock alone needs to know which of the two pages
// that can hold it (order-confirmation vs order-lookup) it's on -- same
// distinction storefront-frontend's own BlockRenderer already makes.
//
// categories powers FeaturedProducts' category picker field -- fetched
// once by EditPage on mount (a real API call, so it can't happen inside
// this synchronous function) and passed in; empty outside the editor
// (Render never reads a component's `fields`, only `render`, so it's
// irrelevant to the live storefront).
export function buildPuckConfig(slug: PageSlug, categories: { id: string; name: string }[] = []): Config {
  return {
    components: {
      Hero: {
        fields: {
          heading: { type: 'text', label: 'Heading' },
          subheading: { type: 'text', label: 'Subheading' },
          imageUrl: { type: 'text', label: 'Image URL' },
          ctaText: { type: 'text', label: 'Button text' },
          ctaUrl: { type: 'text', label: 'Button link' },
        },
        defaultProps: { heading: 'Super Promoção até 60% off', subheading: '', imageUrl: '', ctaText: 'Ver promoções', ctaUrl: '' },
        render: ({ heading, subheading, imageUrl, ctaText, ctaUrl }) => (
          <div
            className="flex min-h-[240px] w-full flex-col justify-center gap-2 bg-brand-50 px-8 py-12"
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
        ),
      },
      FeaturedProducts: {
        fields: {
          title: { type: 'text', label: 'Title' },
          subtitle: { type: 'text', label: 'Subtitle' },
          categoryId: {
            type: 'select',
            label: 'Category (ignored if Product IDs below is set)',
            options: [{ label: 'All categories', value: '' }, ...categories.map(c => ({ label: c.name, value: c.id }))],
          },
          // Raw comma-separated ids rather than a searchable picker --
          // a real product-search picker inside Puck's sidebar (via its
          // `type: 'custom'` field) is a real, separate increment, not
          // built yet. A merchant finds ids from the admin product list.
          productIds: { type: 'text', label: 'Product IDs (comma-separated, overrides Category)' },
          excludeProductIds: { type: 'text', label: 'Exclude product IDs (comma-separated)' },
          limit: { type: 'number', label: 'Number of products', min: 1 },
          itemsPerRowDesktop: { type: 'number', label: 'Items per row (desktop)', min: 1, max: 6 },
          itemsPerRowMobile: { type: 'number', label: 'Items per row (mobile)', min: 1, max: 3 },
        },
        defaultProps: {
          title: 'Os queridinhos do Brasil', subtitle: '', categoryId: '', productIds: '', excludeProductIds: '',
          limit: 8, itemsPerRowDesktop: 4, itemsPerRowMobile: 2,
        },
        render: ({ title, subtitle, categoryId, productIds, excludeProductIds, limit, itemsPerRowDesktop, itemsPerRowMobile }) => (
          <FeaturedProductsBlock
            title={title}
            subtitle={subtitle || undefined}
            categoryId={categoryId || undefined}
            productIds={parseIdList(productIds)}
            excludeProductIds={parseIdList(excludeProductIds)}
            limit={typeof limit === 'number' ? limit : 8}
            itemsPerRowDesktop={typeof itemsPerRowDesktop === 'number' ? itemsPerRowDesktop : 4}
            itemsPerRowMobile={typeof itemsPerRowMobile === 'number' ? itemsPerRowMobile : 2}
          />
        ),
      },
      RichText: {
        fields: { content: { type: 'textarea', label: 'Text' } },
        defaultProps: { content: 'Edit this text…' },
        render: ({ content }) => (
          <div className="w-full whitespace-pre-wrap px-4 py-6 text-brand-700">{content}</div>
        ),
      },
      CustomHTML: {
        fields: { html: { type: 'textarea', label: 'HTML' } },
        defaultProps: { html: '' },
        // Admin-authored (edit-session-token-gated, never shopper input),
        // same trust boundary as the freeform canvas's own custom_html block.
        render: ({ html }) => <div dangerouslySetInnerHTML={{ __html: html }} />,
      },
      ProductGrid: {
        fields: {},
        render: () => <ProductGridBlock />,
      },
      ProductDetails: {
        fields: {},
        render: () => <ProductDetailsBlock />,
      },
      CartSummary: {
        fields: {},
        render: () => <CartSummaryBlock />,
      },
      CheckoutForm: {
        fields: {},
        render: () => <CheckoutFormBlock />,
      },
      OrderStatus: {
        fields: {},
        render: () => <OrderStatusBlock pageSlug={slug === 'order-lookup' ? 'order-lookup' : 'order-confirmation'} />,
      },
      Wishlist: {
        fields: {},
        render: () => <WishlistBlock />,
      },
      LoginForm: {
        fields: {},
        render: () => <LoginFormBlock />,
      },
      Account: {
        fields: {},
        render: () => <AccountBlock />,
      },
      NavMenu: {
        fields: {
          links: {
            type: 'array',
            label: 'Links',
            arrayFields: {
              label: { type: 'text', label: 'Label' },
              url: { type: 'text', label: 'URL (an internal path like /products, or a full https:// link)' },
            },
            defaultItemProps: { label: 'Link', url: '/' },
            getItemSummary: (item: NavMenuLink) => item.label || 'Link',
          },
        },
        defaultProps: { links: [] },
        render: ({ links }) => <NavMenuBlock links={(links as NavMenuLink[] | undefined) ?? []} />,
      },
      NavActions: {
        fields: {},
        render: () => <NavActionsBlock />,
      },
    },
  }
}
