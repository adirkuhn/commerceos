import { Suspense, lazy } from 'react'
import type { Data } from '@puckeditor/core'
import { useThemePage } from './useThemePage'
import { BlockRenderer } from './BlockRenderer'
import { useIsMobileViewport, defaultMobileRect, MOBILE_CANVAS_WIDTH } from './mobileLayout'
import { REQUIRED_BLOCK_TYPE_BY_SLUG } from '../types'
import type { Block, PageSlug, Rect, ThemePage } from '../types'

// Lazy: @puckeditor/core's whole runtime (dnd-kit, tiptap, etc) must never
// be in the bundle a shopper downloads just to view a freeform page --
// only fetched the first time a page actually resolves to editor_mode
// 'puck'. See PuckPageRenderer.tsx's own doc comment.
const PuckPageRenderer = lazy(() => import('../puck/PuckPageRenderer'))

const EMPTY_PUCK_DATA: Data = { content: [], root: {} }

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

export function CanvasPage({ slug }: Props) {
  const { page, loading, error } = useThemePage(slug)
  const isMobile = useIsMobileViewport()

  if (loading) {
    return <p className="px-6 py-16 text-center text-brand-500">Loading…</p>
  }
  if (error) {
    return <p className="px-6 py-16 text-center text-red-600">Could not load this page: {error}</p>
  }
  if (!page) {
    return null
  }

  // Puck-authored pages (editor_mode 'puck', see frontend/theme/storefrontLink.ts's
  // "Edit visually" launcher) render through Puck's own read-only Render
  // instead of the freeform absolutely-positioned block list below --
  // puck_data is a wholly separate document, not translated into Block[].
  if (page.editor_mode === 'puck') {
    return (
      <Suspense fallback={<p className="px-6 py-16 text-center text-brand-500">Loading…</p>}>
        <PuckPageRenderer slug={slug} data={(page.puck_data as Data | undefined) ?? EMPTY_PUCK_DATA} />
      </Suspense>
    )
  }

  const blocks = withRequiredBlock(page)
  if (blocks.length === 0) {
    return <p className="px-6 py-16 text-center text-brand-500">This page has no content yet.</p>
  }

  const rectFor = (block: Block): Rect => (isMobile ? block.layout.mobile ?? defaultMobileRect(block.layout.desktop) : block.layout.desktop)
  const canvasWidth = isMobile ? MOBILE_CANVAS_WIDTH : page.canvas_width
  const height = Math.max(...blocks.map(b => { const r = rectFor(b); return r.y + r.height }), 400)

  return (
    <div className="overflow-x-auto">
      {page.custom_css && <style>{page.custom_css}</style>}
      <div
        data-testid="canvas-page"
        className="relative mx-auto"
        style={{ width: canvasWidth, height }}
      >
        {blocks.map(block => {
          const rect = rectFor(block)
          return (
            <div
              key={block.id}
              className="absolute"
              style={{ left: rect.x, top: rect.y, width: rect.width, height: rect.height }}
            >
              <BlockRenderer block={block} pageSlug={page.slug} />
            </div>
          )
        })}
      </div>
    </div>
  )
}
