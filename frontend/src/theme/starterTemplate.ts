import { createBlock } from './blockDefaults'
import type { Block } from './types'

export interface StarterCategory {
  id: string
  name: string
}

// Real category sections cap at this many -- a merchant with dozens of
// categories doesn't want dozens of stacked Featured Products blocks by
// default, just a realistic multi-section starting layout they can trim
// or extend. Same cap used for the header nav.
const MAX_STARTER_CATEGORY_SECTIONS = 6

// buildStarterHomeBlocks gives a brand-new internal channel's home page a
// real starting point: hero, an about/story blurb, then one Featured
// Products section per existing category (the tenant's own real catalog
// structure, not fabricated content -- mirrors a typical grocery/retail
// site's layout of several category-based sections) capped at
// MAX_STARTER_CATEGORY_SECTIONS, or a single generic product grid when
// there are no categories yet to split by. A merchant can still freely
// move, resize, or delete any of these afterward, same as a block added
// by hand. Site-wide chrome (nav/footer) isn't part of this page any
// more -- see buildStarterHeaderBlocks/buildStarterFooterBlocks, seeded
// onto their own header/footer pseudo-pages instead (see backend
// domain/theme.PageSlugHeader's own doc comment).
export function buildStarterHomeBlocks(storeName: string, categories: StarterCategory[] = []): Block[] {
  const hero = createBlock('hero')
  hero.layout.desktop = { x: 0, y: 0, width: 1440, height: 400 }
  hero.props = {
    heading: storeName || 'Welcome to our store',
    subheading: 'Quality products, delivered with care.',
    imageUrl: '',
  }

  const about = createBlock('rich_text')
  about.layout.desktop = { x: 0, y: 420, width: 1440, height: 160 }
  about.props = {
    content: `${storeName || 'Our store'} brings you the products you love, sourced with care and shipped straight to your door.`,
  }

  const sections = categories.slice(0, MAX_STARTER_CATEGORY_SECTIONS)
  if (sections.length === 0) {
    const products = createBlock('product_grid')
    products.layout.desktop = { x: 0, y: 600, width: 1440, height: 600 }
    return [hero, about, products]
  }

  const featuredBlocks = sections.map((category, i) => {
    const featured = createBlock('featured_products')
    featured.layout.desktop = { x: 0, y: 600 + i * 440, width: 1440, height: 420 }
    featured.props = { ...featured.props, title: category.name, categoryId: category.id }
    return featured
  })

  return [hero, about, ...featuredBlocks]
}

// buildStarterHeaderBlocks gives a brand-new internal channel's header
// page a real starting point: the store name and a nav menu -- linking
// to the tenant's own real categories (their own catalog structure, not
// fabricated content) when any exist, capped at
// MAX_STARTER_CATEGORY_SECTIONS same as the home page's sections, or two
// generic universally-sensible links when there are no categories yet.
// nav_actions (Cart/Wishlist/Account) is header's one required block and
// needs no seeding, storefront-frontend always synthesizes it even
// before a merchant ever opens the editor (see storefront-frontend
// CanvasPage.withRequiredBlock).
export function buildStarterHeaderBlocks(storeName: string, categories: StarterCategory[] = []): Block[] {
  const logo = createBlock('hero')
  logo.layout.desktop = { x: 0, y: 0, width: 300, height: 48 }
  logo.props = { heading: storeName || 'Store', subheading: '', imageUrl: '', ctaText: '', ctaUrl: '' }

  const links = categories.slice(0, MAX_STARTER_CATEGORY_SECTIONS).map(c => ({ label: c.name, url: `/products?category=${c.id}` }))

  const nav = createBlock('nav_menu')
  nav.layout.desktop = { x: 320, y: 0, width: 700, height: 48 }
  nav.props = { links: links.length > 0 ? links : [{ label: 'Products', url: '/products' }, { label: 'Track order', url: '/order-lookup' }] }

  return [logo, nav]
}

// buildStarterFooterBlocks gives a brand-new internal channel's footer
// page a real starting point matching a typical storefront footer's
// layout -- a tagline, a legal/contact link row, and a contact-info
// block -- all generic placeholder copy a merchant fills in with their
// own real details, not baked with any specific real business' info,
// since this same seed function runs for every tenant's every new
// internal channel on this multi-tenant deployment.
export function buildStarterFooterBlocks(storeName: string): Block[] {
  const tagline = createBlock('rich_text')
  tagline.layout.desktop = { x: 0, y: 0, width: 1440, height: 60 }
  tagline.props = { content: `${storeName || 'Our store'} -- quality products, delivered with care.` }

  const legalNav = createBlock('nav_menu')
  legalNav.layout.desktop = { x: 0, y: 80, width: 600, height: 40 }
  legalNav.props = {
    links: [
      { label: 'Privacy policy', url: '/' },
      { label: 'Terms & conditions', url: '/' },
      { label: 'Returns & refunds', url: '/' },
    ],
  }

  const info = createBlock('custom_html')
  info.layout.desktop = { x: 0, y: 140, width: 1440, height: 160 }
  info.props = { html: footerInfoHTML(storeName) }

  return [tagline, legalNav, info]
}

function footerInfoHTML(storeName: string): string {
  const name = storeName || 'Our store'
  return `<div style="display:flex;flex-wrap:wrap;gap:32px;justify-content:center;padding:32px 16px;border-top:1px solid #e5e7eb;font-family:system-ui,sans-serif;color:#4b5563;font-size:14px;">
  <div>
    <p style="font-weight:600;margin:0 0 8px;color:#111827;">${name}</p>
    <p style="margin:0;">Street address, City</p>
    <p style="margin:0;">Open 10:00 - 18:00</p>
  </div>
  <div>
    <p style="font-weight:600;margin:0 0 8px;color:#111827;">Contact</p>
    <p style="margin:0;">email@example.com</p>
    <p style="margin:0;">+31 6 00000000</p>
  </div>
  <div>
    <p style="font-weight:600;margin:0 0 8px;color:#111827;">Follow us</p>
    <p style="margin:0;">Instagram &middot; Facebook &middot; WhatsApp</p>
  </div>
</div>`
}

// ---------------------------------------------------------------------------
// Editorial template -- a second starter layout, picked instead of the
// classic one above (see STARTER_TEMPLATES). Same block types, same
// category-driven content rules (real catalog structure, generic
// placeholder copy elsewhere), but a distinct arrangement: a taller
// full-bleed hero, a centered narrow story column instead of a full-width
// one, and category sections in a two-up grid instead of stacked full-width
// rows -- reads as a boutique/editorial storefront rather than a grocery-
// style category list.
const EDITORIAL_COLUMN_WIDTH = 700
const EDITORIAL_COLUMN_GAP = 40

export function buildEditorialHomeBlocks(storeName: string, categories: StarterCategory[] = []): Block[] {
  const hero = createBlock('hero')
  hero.layout.desktop = { x: 0, y: 0, width: 1440, height: 520 }
  hero.props = {
    heading: storeName || 'Welcome to our store',
    subheading: 'Curated picks, thoughtfully sourced.',
    imageUrl: '',
  }

  const about = createBlock('rich_text')
  about.layout.desktop = { x: 220, y: 540, width: 1000, height: 140 }
  about.props = {
    content: `${storeName || 'Our store'} brings you the products you love, sourced with care and shipped straight to your door.`,
  }

  const sections = categories.slice(0, MAX_STARTER_CATEGORY_SECTIONS)
  if (sections.length === 0) {
    const products = createBlock('product_grid')
    products.layout.desktop = { x: 0, y: 700, width: 1440, height: 600 }
    return [hero, about, products]
  }

  const featuredBlocks = sections.map((category, i) => {
    const featured = createBlock('featured_products')
    const column = i % 2
    featured.layout.desktop = {
      x: column === 0 ? 0 : EDITORIAL_COLUMN_WIDTH + EDITORIAL_COLUMN_GAP,
      y: 700 + Math.floor(i / 2) * 440,
      width: EDITORIAL_COLUMN_WIDTH,
      height: 420,
    }
    featured.props = { ...featured.props, title: category.name, categoryId: category.id }
    return featured
  })

  return [hero, about, ...featuredBlocks]
}

export function buildEditorialHeaderBlocks(storeName: string, categories: StarterCategory[] = []): Block[] {
  const logo = createBlock('hero')
  logo.layout.desktop = { x: 570, y: 0, width: 300, height: 32 }
  logo.props = { heading: storeName || 'Store', subheading: '', imageUrl: '', ctaText: '', ctaUrl: '' }

  const links = categories.slice(0, MAX_STARTER_CATEGORY_SECTIONS).map(c => ({ label: c.name, url: `/products?category=${c.id}` }))

  const nav = createBlock('nav_menu')
  nav.layout.desktop = { x: 320, y: 40, width: 800, height: 32 }
  nav.props = { links: links.length > 0 ? links : [{ label: 'Products', url: '/products' }, { label: 'Track order', url: '/order-lookup' }] }

  return [logo, nav]
}

export function buildEditorialFooterBlocks(storeName: string): Block[] {
  const tagline = createBlock('rich_text')
  tagline.layout.desktop = { x: 220, y: 0, width: 1000, height: 60 }
  tagline.props = { content: `${storeName || 'Our store'} -- quality products, delivered with care.` }

  const newsletter = createBlock('custom_html')
  newsletter.layout.desktop = { x: 0, y: 80, width: 1440, height: 120 }
  newsletter.props = { html: editorialNewsletterHTML() }

  const legalNav = createBlock('nav_menu')
  legalNav.layout.desktop = { x: 420, y: 220, width: 600, height: 40 }
  legalNav.props = {
    links: [
      { label: 'Privacy policy', url: '/' },
      { label: 'Terms & conditions', url: '/' },
      { label: 'Returns & refunds', url: '/' },
    ],
  }

  return [tagline, newsletter, legalNav]
}

function editorialNewsletterHTML(): string {
  return `<div style="text-align:center;padding:32px 16px;border-top:1px solid #e5e7eb;border-bottom:1px solid #e5e7eb;font-family:system-ui,sans-serif;">
  <p style="font-weight:600;margin:0 0 4px;color:#111827;">Join our newsletter</p>
  <p style="margin:0 0 12px;color:#4b5563;font-size:14px;">New arrivals and offers, straight to your inbox.</p>
  <p style="margin:0;color:#9ca3af;font-size:12px;">email@example.com</p>
</div>`
}

export type StarterTemplateId = 'classic' | 'editorial'

export interface StarterTemplate {
  id: StarterTemplateId
  label: string
  buildHeader: (storeName: string, categories?: StarterCategory[]) => Block[]
  buildHome: (storeName: string, categories?: StarterCategory[]) => Block[]
  buildFooter: (storeName: string) => Block[]
}

// STARTER_TEMPLATES is the picker's source of truth -- both the internal-
// channel create form (SalesChannels.tsx) and the theme editor's own
// "Template" dropdown (ThemeEditorPage.tsx) render from this list rather
// than hard-coding template names, so a third template only needs adding
// here.
export const STARTER_TEMPLATES: StarterTemplate[] = [
  { id: 'classic', label: 'Classic', buildHeader: buildStarterHeaderBlocks, buildHome: buildStarterHomeBlocks, buildFooter: buildStarterFooterBlocks },
  { id: 'editorial', label: 'Editorial', buildHeader: buildEditorialHeaderBlocks, buildHome: buildEditorialHomeBlocks, buildFooter: buildEditorialFooterBlocks },
]

// Block components (BlockRenderer in storefront-frontend) render fixed
// Tailwind markup regardless of which starter template built the block --
// hero/featured_products headings are always plain-sans h2s, cards are
// always .rounded-lg, no per-template style hook exists on the components
// themselves. So a template's only real visual levers are (a) block
// position/composition (see build*Blocks above) and (b) custom_css, a raw
// <style> tag every page already supports (see palettes.ts). Classic's
// entry here is '' -- the plain look every internal channel has always
// had, unchanged. Editorial overrides headings to a serif face and softens
// every rounded corner site-wide, so the two templates read as visually
// distinct even for a tenant with no categories yet (where the two
// layouts' block positions alone are barely different). !important is
// necessary, not sloppy: Tailwind's own utility classes (equal or higher
// selector specificity, e.g. .rounded-lg) come from a stylesheet loaded
// before this inline tag and would otherwise win the cascade.
export const STARTER_TEMPLATE_CSS: Record<StarterTemplateId, string> = {
  classic: '',
  editorial: `h1, h2 {\n  font-family: Georgia, 'Times New Roman', serif !important;\n  letter-spacing: 0.01em !important;\n}\n.rounded, .rounded-lg {\n  border-radius: 1rem !important;\n}`,
}

// combinedStarterCSS is what every seeded/reapplied starter page's
// custom_css is actually set to -- the palette's :root color variables
// (see paletteCustomCSS) plus the chosen template's own overrides above,
// concatenated into the one <style> tag a ThemePage carries.
export function combinedStarterCSS(paletteCSS: string, templateId: StarterTemplateId): string {
  const templateCSS = STARTER_TEMPLATE_CSS[templateId]
  return templateCSS ? `${paletteCSS}\n\n${templateCSS}` : paletteCSS
}
