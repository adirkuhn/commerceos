import type { SitemapData } from '../api/storefront'
import type { StorefrontTheme } from '../core/document'
import type { StoreLanguages } from './storeLocale'

// The store's sitemap.xml: home, the live theme's pages, then the
// backend's collections and products (which carry when they changed) --
// each in every language the store offers.

const escapeXML = (s: string) => s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' })[c]!)

export function sitemapXML(origin: string, storeKey: string, theme: StorefrontTheme, data: SitemapData, langs?: StoreLanguages): string {
  const store = `${origin.replace(/\/+$/, '')}/store/${encodeURIComponent(storeKey)}/`
  const bases = (langs?.offered ?? []).filter(l => l !== langs?.defaultLocale).map(l => `${store}${l}/`)
  const url = (path: string, lastmod?: string) => [store, ...bases].map(base =>
    `<url><loc>${escapeXML(base + path)}</loc>${lastmod ? `<lastmod>${escapeXML(lastmod)}</lastmod>` : ''}</url>`).join('\n')
  const entries = [
    url(''),
    ...theme.documents.filter(d => d.kind === 'page').map(d => url(`pages/${encodeURIComponent(d.key)}`)),
    ...data.collections.map(c => url(`collections/${encodeURIComponent(c.slug)}`, c.updated_at)),
    ...data.products.map(p => url(`products/${encodeURIComponent(p.slug)}`, p.updated_at)),
  ]
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${entries.join('\n')}\n</urlset>\n`
}
