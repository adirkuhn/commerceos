import { Suspense, lazy } from 'react'
import type { Data } from '@puckeditor/core'
import { useThemePage } from './useThemePage'
import { BlockRenderer } from './BlockRenderer'
import { useIsMobileViewport, defaultMobileRect, MOBILE_CANVAS_WIDTH } from './mobileLayout'
import { REQUIRED_BLOCK_TYPE_BY_SLUG } from '../types'
import type { Block, BlockType, PageSlug, Rect, ThemePage } from '../types'

// DEFAULT_REQUIRED_BLOCK_HEIGHT sizes a synthesized required block
// sensibly for its own context -- nav_actions is a slim header nav row
// sitting alongside the logo/nav_menu blocks (see PageSlugHeader's own
// doc comment), not a full page section like every other required block
// type (cart/checkout/product/order/wishlist/login/account), which all
// use DEFAULT_SYNTHESIZED_HEIGHT.
const DEFAULT_REQUIRED_BLOCK_HEIGHT: Partial<Record<BlockType, number>> = {
  nav_actions: 48,
}
const DEFAULT_SYNTHESIZED_HEIGHT = 600

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
  const height = DEFAULT_REQUIRED_BLOCK_HEIGHT[required] ?? DEFAULT_SYNTHESIZED_HEIGHT
  return [
    ...page.blocks,
    { id: `default-${required}`, type: required, layout: { desktop: { x: 0, y, width: page.canvas_width, height }, mobile: null }, props: {} },
  ]
}

// isChrome pages (header/footer) render around every real page rather
// than being navigated to directly -- a full-page loading/error/empty
// message (sized for a shopper who just clicked a link to get here)
// would be jarring chrome flashing on every visit, so they collapse to
// nothing instead: the page's own real content is still the useful
// thing on screen underneath/around, not the pseudo-page's own state.
function isChrome(slug: PageSlug): boolean {
  return slug === 'header' || slug === 'footer'
}

export function CanvasPage({ slug }: Props) {
  const { page, loading, error } = useThemePage(slug)
  const isMobile = useIsMobileViewport()
  const chrome = isChrome(slug)

  if (loading) {
    return chrome ? null : <p className="px-6 py-16 text-center text-brand-500">Loading…</p>
  }
  if (error) {
    return chrome ? null : <p className="px-6 py-16 text-center text-red-600">Could not load this page: {error}</p>
  }
  if (!page) {
    return null
  }

  // Puck-authored pages (editor_mode 'puck', see frontend/theme/storefrontLink.ts's
  // "Edit visually" launcher) render through Puck's own read-only Render
  // instead of the freeform absolutely-positioned block list below --
  // puck_data is a wholly separate document, not translated into Block[].
  // custom_css (palette colors, a starter template's own typography
  // overrides -- see frontend/theme/starterTemplate.ts) is a property of
  // the page itself, independent of which document is active, so it must
  // render in both branches -- this one used to return before ever
  // reaching the <style> tag below, silently dropping every Puck-mode
  // page's palette/template styling entirely.
  if (page.editor_mode === 'puck') {
    return (
      <Suspense fallback={<p className="px-6 py-16 text-center text-brand-500">Loading…</p>}>
        {page.custom_css && <style>{page.custom_css}</style>}
        <PuckPageRenderer slug={slug} data={(page.puck_data as Data | undefined) ?? EMPTY_PUCK_DATA} />
      </Suspense>
    )
  }

  const blocks = withRequiredBlock(page)
  if (blocks.length === 0) {
    return chrome ? null : <p className="px-6 py-16 text-center text-brand-500">This page has no content yet.</p>
  }

  const rectFor = (block: Block): Rect => (isMobile ? block.layout.mobile ?? defaultMobileRect(block.layout.desktop) : block.layout.desktop)
  const canvasWidth = isMobile ? MOBILE_CANVAS_WIDTH : page.canvas_width
  // Chrome (header/footer) sizes to its actual content, no artificial
  // floor -- a 400px-tall header/footer on every single page would be
  // its own, worse version of this same bug. A real page still gets the
  // 400px floor so a nearly-empty one doesn't look cut off.
  const height = Math.max(...blocks.map(b => { const r = rectFor(b); return r.y + r.height }), chrome ? 0 : 400)

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
