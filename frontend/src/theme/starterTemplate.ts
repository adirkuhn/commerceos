import { createBlock } from './blockDefaults'
import type { Block } from './types'

// buildStarterHomeBlocks gives a brand-new internal channel's home page a
// real starting point (hero, an about/story blurb, the product catalog,
// a footer-style info block) instead of the blank canvas
// GetThemePageUseCase falls back to -- a merchant can still freely move,
// resize, or delete any of these afterward, same as a block added by hand.
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

  const footer = createBlock('custom_html')
  footer.layout.desktop = { x: 0, y: 1220, width: 1440, height: 220 }
  footer.props = {
    html: footerHTML(storeName),
  }

  return [hero, about, products, footer]
}

function footerHTML(storeName: string): string {
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
