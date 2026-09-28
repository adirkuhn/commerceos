import { describe, it, expect, vi } from 'vitest'
import { render } from '../entry-server'
import { defaultSettings } from '../../core/settings'

const theme = {
  version_id: 'v1', version_number: 3, preview: false, settings: defaultSettings(),
  documents: [
    { kind: 'template', key: 'home', title: '' }, { kind: 'template', key: 'not-found', title: '' },
    { kind: 'section_group', key: 'header', title: '' }, { kind: 'section_group', key: 'footer', title: '' },
    { kind: 'page', key: 'about', title: 'About us' },
  ],
}

const withGrid = (d: ReturnType<typeof baseDoc>) => d.key === 'home'
  ? { ...d, root: { ...d.root, children: [...d.root.children, { id: 'g', type: 'product-grid', props: { source: 'all', limit: 4, show_price: true } }] } }
  : d

const doc = (kind: string, key: string, text: string) => withGrid(baseDoc(kind, key, text))

const baseDoc = (kind: string, key: string, text: string) => ({
  kind, key, title: key === 'about' ? 'About us' : '', seo: key === 'home' ? { title: 'Shop home', description: 'Best <shop>' } : {},
  root: { id: 'root', type: 'root', children: [{ id: `h-${key}`, type: 'heading', props: { text, level: 1 } }] },
})

function backend(opts: { themeStatus?: number } = {}) {
  return vi.fn<typeof fetch>(async input => {
    const url = String(input)
    const json = (data: unknown, status = 200) => new Response(JSON.stringify(status === 200 ? { data, error: null } : { data: null, error: { code: 'NOT_FOUND', message: 'x' } }), { status })
    if (url.endsWith('/storefront/v1/theme')) return json(theme, opts.themeStatus ?? 200)
    if (url.includes('/storefront/v1/products?')) {
      return json({ items: [{ id: 'p1', name: 'Red shoes', slug: 'red-shoes', category_id: null, tags: [], image_url: null, price_minor_units: 2999, currency_code: 'EUR' }], total: 1 })
    }
    const m = /documents\/([^/]+)\/([^/]+)$/.exec(url)
    if (m) return json(doc(m[1], decodeURIComponent(m[2]), `${m[2]} text`))
    return json(null, 404)
  })
}

const base = { apiBaseURL: 'http://backend/api/v1', publicApiBaseURL: 'https://api.test/api/v1' }

describe('entry-server render', () => {
  it('renders header, template and footer for the channel with SEO head', async () => {
    const out = await render({ ...base, url: '/store/key-1/', fetch: backend() })

    expect(out.status).toBe(200)
    expect(out.html).toContain('header text')
    expect(out.html).toContain('home text')
    expect(out.html).toContain('footer text')
    expect(out.head).toContain('<title>Shop home</title>')
    expect(out.head).toContain('content="Best &lt;shop&gt;"')
    expect(out.head).toContain('--color-primary:#1f3fbf')
    expect(out.state).toContain('"storeKey":"key-1"')
    expect(out.state).not.toContain('<')
  })

  it('prefetches catalog data so product grids are server-rendered and hydrate from cache', async () => {
    const f = backend()
    const out = await render({ ...base, url: '/store/key-1/', fetch: f })
    expect(out.html).toContain('Red shoes')
    expect(out.html).toContain('href="/store/key-1/products/red-shoes"')
    expect(out.state).toContain('Red shoes')
  })

  it('links the favicon chosen in theme settings', async () => {
    const fav = '0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e'
    theme.settings = { ...theme.settings, brand: { favicon_asset_id: fav } }
    ;(theme as Record<string, unknown>).assets = { [fav]: { url: 'https://cdn.test/fav.png', width: 32, height: 32, alt: '' } }
    try {
      const out = await render({ ...base, url: '/store/key-1/', fetch: backend() })
      expect(out.head).toContain('<link rel="icon" href="https://cdn.test/fav.png">')
    } finally {
      theme.settings = defaultSettings()
      delete (theme as Record<string, unknown>).assets
    }
  })

  it('renders merchant pages', async () => {
    const out = await render({ ...base, url: '/store/key-1/pages/about', fetch: backend() })
    expect(out.status).toBe(200)
    expect(out.html).toContain('about text')
    expect(out.head).toContain('<title>About us</title>')
  })

  it('renders the not-found template with a 404 for unknown paths and missing pages', async () => {
    for (const url of ['/store/key-1/nope', '/store/key-1/pages/missing']) {
      const out = await render({ ...base, url, fetch: backend() })
      expect(out.status, url).toBe(404)
      expect(out.html, url).toContain('not-found text')
    }
  })

  it('shows a plain unavailable page when the channel has no live theme', async () => {
    const out = await render({ ...base, url: '/store/key-1/', fetch: backend({ themeStatus: 404 }) })
    expect(out.status).toBe(404)
    expect(out.html).toContain('This store is not available')
  })

  it('passes the preview token to the API and marks the page as a preview', async () => {
    const f = backend()
    const out = await render({ ...base, url: '/store/key-1/', previewToken: 'pv', fetch: f })
    expect(new Headers(f.mock.calls[0][1]?.headers).get('X-Theme-Preview')).toBe('pv')
    expect(out.head).toContain('noindex')
  })
})
