import { useThemePage } from './useThemePage'
import { BlockRenderer } from './BlockRenderer'
import type { PageSlug } from '../types'

interface Props {
  slug: PageSlug
}

// Desktop-only for now (see ThemePage.Layout.Mobile on the backend --
// present but not yet populated), so every block renders at its saved
// desktop Rect regardless of viewport. A canvas with no saved blocks yet
// (a brand-new channel) is a normal, not an error, state.
export function CanvasPage({ slug }: Props) {
  const { page, loading, error } = useThemePage(slug)

  if (loading) {
    return <p className="px-6 py-16 text-center text-brand-500">Loading…</p>
  }
  if (error) {
    return <p className="px-6 py-16 text-center text-red-600">Could not load this page: {error}</p>
  }
  if (!page || page.blocks.length === 0) {
    return <p className="px-6 py-16 text-center text-brand-500">This page has no content yet.</p>
  }

  const height = Math.max(...page.blocks.map(b => b.layout.desktop.y + b.layout.desktop.height), 400)

  return (
    <div className="overflow-x-auto">
      <div
        data-testid="canvas-page"
        className="relative mx-auto"
        style={{ width: page.canvas_width, height }}
      >
        {page.blocks.map(block => (
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
            <BlockRenderer block={block} />
          </div>
        ))}
      </div>
    </div>
  )
}
