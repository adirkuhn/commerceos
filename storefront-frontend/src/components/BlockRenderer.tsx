import type { Block } from '../types'

function stringProp(props: Record<string, unknown>, key: string): string {
  const v = props[key]
  return typeof v === 'string' ? v : ''
}

const FUNCTIONAL_LABELS: Record<string, string> = {
  product_grid: 'Product grid',
  product_details: 'Product details',
  cart_summary: 'Cart summary',
  checkout_form: 'Checkout form',
  order_status: 'Order status',
}

// Renders one block's real content. hero/rich_text/custom_html render
// merchant-authored props directly; the rest are wired to real cart/
// catalog/checkout data in a later increment -- until then they render a
// labeled placeholder rather than pretending to be the finished feature.
export function BlockRenderer({ block }: { block: Block }) {
  switch (block.type) {
    case 'hero': {
      const heading = stringProp(block.props, 'heading')
      const subheading = stringProp(block.props, 'subheading')
      const imageUrl = stringProp(block.props, 'imageUrl')
      return (
        <div
          className="flex h-full w-full flex-col justify-center gap-2 bg-brand-50 px-8"
          style={imageUrl ? { backgroundImage: `url(${imageUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : undefined}
        >
          {heading && <h2 className="text-3xl font-semibold text-brand-700">{heading}</h2>}
          {subheading && <p className="text-brand-500">{subheading}</p>}
        </div>
      )
    }
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
    default:
      return (
        <div className="flex h-full w-full items-center justify-center border border-dashed border-gray-300 bg-gray-50">
          <p className="text-sm text-gray-400">{FUNCTIONAL_LABELS[block.type] ?? block.type} — coming soon</p>
        </div>
      )
  }
}
