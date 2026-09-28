import { describe, it, expect } from 'vitest'
import { sitemapXML } from '../sitemap'
import type { StorefrontTheme } from '../../core/document'
import { defaultSettings } from '../../core/settings'

const theme: StorefrontTheme = {
  version_id: 'v1', version_number: 1, preview: false, settings: defaultSettings(),
  documents: [
    { kind: 'template', key: 'home', title: '' }, { kind: 'page', key: 'about', title: 'About' },
    { kind: 'page', key: 'faq', title: 'FAQ' }, { kind: 'section_group', key: 'header', title: '' },
  ],
}

describe('sitemapXML', () => {
  it('lists home, theme pages, collections and products with absolute URLs', () => {
    const xml = sitemapXML('https://shop.test/', 'key 1', theme, {
      products: [{ slug: 'red-shoes', updated_at: '2026-09-01T12:00:00Z' }],
      collections: [{ slug: 'summer & sun', updated_at: '2026-08-01T00:00:00Z' }],
    })
    expect(xml.startsWith('<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">')).toBe(true)
    const locs = [...xml.matchAll(/<loc>(.*?)<\/loc>/g)].map(m => m[1])
    expect(locs).toEqual([
      'https://shop.test/store/key%201/',
      'https://shop.test/store/key%201/pages/about',
      'https://shop.test/store/key%201/pages/faq',
      'https://shop.test/store/key%201/collections/summer%20%26%20sun',
      'https://shop.test/store/key%201/products/red-shoes',
    ])
    expect(xml).toContain('<url><loc>https://shop.test/store/key%201/products/red-shoes</loc><lastmod>2026-09-01T12:00:00Z</lastmod></url>')
    expect(xml.trimEnd().endsWith('</urlset>')).toBe(true)
  })
})
