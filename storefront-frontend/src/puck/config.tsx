import type { Config } from '@puckeditor/core'
import { ProductGridBlock } from '../components/blocks/ProductGridBlock'
import { ProductDetailsBlock } from '../components/blocks/ProductDetailsBlock'
import { CartSummaryBlock } from '../components/blocks/CartSummaryBlock'
import { CheckoutFormBlock } from '../components/blocks/CheckoutFormBlock'
import { OrderStatusBlock } from '../components/blocks/OrderStatusBlock'
import { WishlistBlock } from '../components/blocks/WishlistBlock'
import { LoginFormBlock } from '../components/blocks/LoginFormBlock'
import { AccountBlock } from '../components/blocks/AccountBlock'
import type { PageSlug } from '../types'

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
export function buildPuckConfig(slug: PageSlug): Config {
  return {
    components: {
      Hero: {
        fields: {
          heading: { type: 'text', label: 'Heading' },
          subheading: { type: 'text', label: 'Subheading' },
          imageUrl: { type: 'text', label: 'Image URL' },
        },
        defaultProps: { heading: 'Welcome', subheading: '', imageUrl: '' },
        render: ({ heading, subheading, imageUrl }) => (
          <div
            className="flex min-h-[240px] w-full flex-col justify-center gap-2 bg-brand-50 px-8 py-12"
            style={imageUrl ? { backgroundImage: `url(${imageUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : undefined}
          >
            {heading && <h2 className="text-3xl font-semibold text-brand-700">{heading}</h2>}
            {subheading && <p className="text-brand-500">{subheading}</p>}
          </div>
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
    },
  }
}
