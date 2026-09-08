import { useThemePage } from './useThemePage'
import { BlockRenderer } from './BlockRenderer'
import { REQUIRED_BLOCK_TYPE_BY_SLUG } from '../types'
import type { Block, PageSlug, ThemePage } from '../types'

interface Props {
  slug: PageSlug
}

// A page whose slug requires a functional block (product/cart/checkout/
// order-confirmation/order-lookup -- see REQUIRED_BLOCK_TYPE_BY_SLUG)
// gets one appended with a full-width default Rect when the saved
// document doesn't have it yet -- real functionality is never missing
// just because a merchant hasn't opened the editor. Home/products have
// no required block: an empty canvas there really does render empty.
function withRequiredBlock(page: ThemePage): Block[] {
  const required = REQUIRED_BLOCK_TYPE_BY_SLUG[page.slug]
  if (!required || page.blocks.some(b => b.type === required)) {
    return page.blocks
  }
  const y = page.blocks.reduce((max, b) => Math.max(max, b.layout.desktop.y + b.layout.desktop.height), 0)
  return [
    ...page.blocks,
    { id: `default-${required}`, type: required, layout: { desktop: { x: 0, y, width: page.canvas_width, height: 600 }, mobile: null }, props: {} },
  ]
}

// Desktop-only for now (see ThemePage.Layout.Mobile on the backend --
// present but not yet populated), so every block renders at its saved
// desktop Rect regardless of viewport.
export function CanvasPage({ slug }: Props) {
  const { page, loading, error } = useThemePage(slug)

  if (loading) {
    return <p className="px-6 py-16 text-center text-brand-500">Loading…</p>
  }
  if (error) {
    return <p className="px-6 py-16 text-center text-red-600">Could not load this page: {error}</p>
  }
  if (!page) {
    return null
  }

  const blocks = withRequiredBlock(page)
  if (blocks.length === 0) {
    return <p className="px-6 py-16 text-center text-brand-500">This page has no content yet.</p>
  }

  const height = Math.max(...blocks.map(b => b.layout.desktop.y + b.layout.desktop.height), 400)

  return (
    <div className="overflow-x-auto">
      <div
        data-testid="canvas-page"
        className="relative mx-auto"
        style={{ width: page.canvas_width, height }}
      >
        {blocks.map(block => (
          <div
            key={block.id}
            className="absolute"
            style={{
              left: block.layout.desktop.x,
              top: block.layout.desktop.y,
              width: block.layout.desktop.width,
              height: block.layout.desktop.height,
            }}
          >
            <BlockRenderer block={block} pageSlug={page.slug} />
          </div>
        ))}
      </div>
    </div>
  )
}
