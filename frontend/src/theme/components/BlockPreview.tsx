import type { Block } from '../types'

// A rough approximation of each block type's real rendering, just enough
// for the merchant to judge layout/positioning while editing -- the
// storefront-frontend owns the real, styled rendering (and, for
// functional blocks, the actual wired-up behavior).
export function BlockPreview({ block }: { block: Block }) {
  switch (block.type) {
    case 'hero': {
      const heading = typeof block.props.heading === 'string' ? block.props.heading : ''
      const subheading = typeof block.props.subheading === 'string' ? block.props.subheading : ''
      return (
        <div className="h-full w-full overflow-hidden bg-brand-50 p-3">
          <p className="truncate text-sm font-semibold text-brand-700">{heading || 'Hero heading'}</p>
          {subheading && <p className="truncate text-xs text-brand-500">{subheading}</p>}
        </div>
      )
    }
    case 'rich_text': {
      const content = typeof block.props.content === 'string' ? block.props.content : ''
      return (
        <div className="h-full w-full overflow-hidden p-3">
          <p className="line-clamp-4 text-xs text-gray-600">{content || 'Rich text block'}</p>
        </div>
      )
    }
    case 'custom_html': {
      const html = typeof block.props.html === 'string' ? block.props.html : ''
      return (
        <div className="h-full w-full overflow-hidden bg-gray-50 p-3">
          <p className="truncate text-[11px] font-mono text-gray-500">{html || '<!-- custom html -->'}</p>
        </div>
      )
    }
    default:
      return (
        <div className="flex h-full w-full items-center justify-center bg-amber-50 p-3">
          <p className="text-[11px] font-medium text-amber-700">{block.type} (functional block)</p>
        </div>
      )
  }
}
