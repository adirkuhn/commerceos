import type { SitemapData } from '../api/storefront'
import type { StorefrontTheme } from '../core/document'

// The store's sitemap.xml: home, the live theme's pages, then the
// backend's collections and products (which carry when they changed).

const escapeXML = (s: string) => s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' })[c]!)

export function sitemapXML(origin: string, storeKey: string, theme: StorefrontTheme, data: SitemapData): string {
  const base = `${origin.replace(/\/+$/, '')}/store/${encodeURIComponent(storeKey)}/`
  const url = (path: string, lastmod?: string) =>
    `<url><loc>${escapeXML(base + path)}</loc>${lastmod ? `<lastmod>${escapeXML(lastmod)}</lastmod>` : ''}</url>`
  const entries = [
    url(''),
    ...theme.documents.filter(d => d.kind === 'page').map(d => url(`pages/${encodeURIComponent(d.key)}`)),
    ...data.collections.map(c => url(`collections/${encodeURIComponent(c.slug)}`, c.updated_at)),
    ...data.products.map(p => url(`products/${encodeURIComponent(p.slug)}`, p.updated_at)),
  ]
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${entries.join('\n')}\n</urlset>\n`
}
