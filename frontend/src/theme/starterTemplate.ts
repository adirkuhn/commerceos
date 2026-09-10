import { createBlock } from './blockDefaults'
import type { Block } from './types'

// buildStarterHomeBlocks gives a brand-new internal channel's home page a
// real starting point (hero, an about/story blurb, the product catalog)
// instead of the blank canvas GetThemePageUseCase falls back to -- a
// merchant can still freely move, resize, or delete any of these
// afterward, same as a block added by hand. Site-wide chrome (nav/footer)
// isn't part of this page any more -- see buildStarterHeaderBlocks/
// buildStarterFooterBlocks, seeded onto their own header/footer
// pseudo-pages instead (see backend domain/theme.PageSlugHeader's own doc
// comment).
export function buildStarterHomeBlocks(storeName: string): Block[] {
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

  const products = createBlock('product_grid')
  products.layout.desktop = { x: 0, y: 600, width: 1440, height: 600 }

  return [hero, about, products]
}

// buildStarterHeaderBlocks gives a brand-new internal channel's header
// page a real starting point: the store name and a nav menu of generic,
// universally-sensible links -- nav_actions (Cart/Wishlist/Account) is
// header's one required block and needs no seeding, storefront-frontend
// always synthesizes it even before a merchant ever opens the editor
// (see storefront-frontend CanvasPage.withRequiredBlock).
export function buildStarterHeaderBlocks(storeName: string): Block[] {
  const logo = createBlock('hero')
  logo.layout.desktop = { x: 0, y: 0, width: 300, height: 48 }
  logo.props = { heading: storeName || 'Store', subheading: '', imageUrl: '', ctaText: '', ctaUrl: '' }

  const nav = createBlock('nav_menu')
  nav.layout.desktop = { x: 320, y: 0, width: 400, height: 48 }
  nav.props = { links: [{ label: 'Products', url: '/products' }, { label: 'Track order', url: '/order-lookup' }] }

  return [logo, nav]
}

// buildStarterFooterBlocks gives a brand-new internal channel's footer
// page a real starting point: a tagline and generic contact/social
// placeholders a merchant fills in with their own real details -- not
// baked with any specific real business' info, since this same seed
// function runs for every tenant's every new internal channel on this
// multi-tenant deployment.
export function buildStarterFooterBlocks(storeName: string): Block[] {
  const tagline = createBlock('rich_text')
  tagline.layout.desktop = { x: 0, y: 0, width: 1440, height: 60 }
  tagline.props = { content: `${storeName || 'Our store'} -- quality products, delivered with care.` }

  const info = createBlock('custom_html')
  info.layout.desktop = { x: 0, y: 80, width: 1440, height: 160 }
  info.props = { html: footerInfoHTML(storeName) }

  return [tagline, info]
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
