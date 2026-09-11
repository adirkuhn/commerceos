import { BLOCK_TYPE_LABELS } from '../types'
import type { Block } from '../types'

// A rough approximation of each block type's real rendering, just enough
// for the merchant to judge layout/positioning while editing -- the
// storefront-frontend owns the real, styled rendering (and, for
// functional blocks, the actual wired-up behavior). Headings use the same
// tag (h2) the live storefront's BlockRenderer uses -- a starter
// template's custom_css (see starterTemplate.ts's STARTER_TEMPLATE_CSS)
// targets h1/h2 to change typography, and ThemeEditorPage now injects
// that same custom_css into this canvas (see its own render), so this
// preview must use matching tags for that override to actually show here,
// not just on the real storefront.
export function BlockPreview({ block }: { block: Block }) {
  switch (block.type) {
    case 'hero': {
      const heading = typeof block.props.heading === 'string' ? block.props.heading : ''
      const subheading = typeof block.props.subheading === 'string' ? block.props.subheading : ''
      return (
        <div className="h-full w-full overflow-hidden bg-brand-50 p-3">
          <h2 className="truncate text-sm font-semibold text-brand-700">{heading || 'Hero heading'}</h2>
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
      // Admin-authored HTML, same trust boundary as the live storefront's
      // own custom_html rendering (see storefront-frontend BlockRenderer's
      // doc comment) -- rendered for real here too, not shown as raw
      // source text, since several starter templates (Editorial, Mercado)
      // use it for real visual content (a footer band, promo banners, a
      // trust-badge row) that a merchant needs to actually see to judge.
      const html = typeof block.props.html === 'string' ? block.props.html : ''
      return html
        ? <div className="h-full w-full overflow-hidden" dangerouslySetInnerHTML={{ __html: html }} />
        : (
          <div className="flex h-full w-full items-center justify-center bg-gray-50 p-3">
            <p className="text-[11px] font-mono text-gray-400">&lt;!-- custom html --&gt;</p>
          </div>
        )
    }
    case 'featured_products': {
      const title = typeof block.props.title === 'string' ? block.props.title : ''
      return (
        <div className="flex h-full w-full items-center justify-center bg-amber-50 p-3">
          {title ? <h2 className="text-[11px] font-medium text-amber-700">{title} — renders live content</h2> : (
            <p className="text-[11px] font-medium text-amber-700">Featured products — renders live content</p>
          )}
        </div>
      )
    }
    case 'nav_menu': {
      const links = Array.isArray(block.props.links) ? block.props.links : []
      return (
        <div className="flex h-full w-full items-center gap-3 overflow-hidden bg-gray-50 px-3">
          {links.length === 0 && <p className="text-xs text-gray-400">Nav menu — no links yet</p>}
          {links.map((l, i) => (
            <p key={i} className="truncate text-xs text-brand-600">
              {typeof l?.label === 'string' && l.label ? l.label : 'Link'}
            </p>
          ))}
        </div>
      )
    }
    default:
      return (
        <div className="flex h-full w-full items-center justify-center bg-amber-50 p-3">
          <p className="text-[11px] font-medium text-amber-700">{BLOCK_TYPE_LABELS[block.type]} — renders live content</p>
        </div>
      )
  }
}
