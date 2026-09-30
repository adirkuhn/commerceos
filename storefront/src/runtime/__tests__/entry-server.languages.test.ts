import { describe, it, expect, vi } from 'vitest'
import { render, renderSitemap } from '../entry-server'
import { defaultSettings } from '../../core/settings'

const theme = {
  version_id: 'v1', version_number: 1, preview: false, settings: defaultSettings(),
  documents: ['home', 'product', 'collection', 'search', 'not-found', 'order-confirmation'].map(key => ({ kind: 'template', key, title: '' })),
}

const tree = (children: unknown[]) => ({ id: 'root', type: 'root', children: [{ id: 's', type: 'section', children }] })
const docs: Record<string, unknown> = {
  product: tree([{ id: 't', type: 'product-title', props: { level: 1 } }, { id: 'pr', type: 'product-price', props: {} },
    { id: 'rv', type: 'product-reviews', props: { title: 'Reviews', limit: 3 } }]),
  collection: tree([{ id: 'h', type: 'collection-header', props: {} }, { id: 'p', type: 'collection-products', props: { per_page: 12 } }, { id: 'pg', type: 'pagination', props: {} }]),
  search: tree([{ id: 'h', type: 'collection-header', props: {} }, { id: 'p', type: 'collection-products', props: {} }]),
  'not-found': tree([{ id: 'nf', type: 'heading', props: { text: 'Not here', level: 1 } }]),
  home: tree([{ id: 'logo', type: 'site-logo', props: { size: 'md' } }]),
  'order-confirmation': tree([{ id: 'os', type: 'order-status', props: { title: 'Thanks!' } }]),
}

const docMeta: Record<string, object> = {}

const productDetail = {
  id: 'p1', name: 'Red shoes', slug: 'red-shoes', description: 'Comfy shoes for walking', category_id: null, tags: [], images: [],
  variants: [{ id: 'v1', name: 'Default', skus: [{ id: 's1', sku_code: 'RS', price_minor_units: 5000, currency_code: 'EUR', stock_quantity: 2 }] }],
}

function backend() {
  // A Portuguese store that also offers English and Dutch.
  const ok = (data: unknown) => new Response(JSON.stringify({ data, error: null }))
  const missing = () => new Response(JSON.stringify({ data: null, error: { code: 'NOT_FOUND', message: 'x' } }), { status: 404 })
  return vi.fn<typeof fetch>(async input => {
    const url = new URL(String(input))
    const p = url.pathname.replace('/api/v1/storefront/v1', '')
    if (p === '/theme') return ok(theme)
    if (p === '/sitemap') return ok({ products: [{ slug: 'red-shoes', updated_at: '2026-09-01T00:00:00Z' }], collections: [] })
    if (p === '/store') return ok({ name: 'Loja', currency_code: 'EUR', locale: 'pt-BR', locales: ['pt-BR', 'en', 'nl'] })
    const doc = /^\/theme\/versions\/v1\/documents\/template\/(.+)$/.exec(p)
    if (doc) return ok({ kind: 'template', key: doc[1], title: '', seo: {}, root: docs[doc[1]], ...docMeta[doc[1]] })
    if (p === '/products/by-slug/red-shoes') return ok(productDetail)
    if (p.startsWith('/products/by-slug/')) return missing()
    if (p === '/products/p1/reviews') return ok({ reviews: [], summary: { average: 0, count: 0 } })
    if (p === '/collections/by-slug/summer') return ok({ id: 'col-1', name: 'Summer', slug: 'summer', description: 'Hot', seo_title: 'Summer sale', seo_description: 'Deals', image_url: null })
    if (p.startsWith('/collections/by-slug/')) return missing()
    if (p === '/products') {
      return ok({ items: [{ id: 'p1', name: 'Red shoes', slug: 'red-shoes', category_id: null, tags: [], image_url: null, price_minor_units: 5000, currency_code: 'EUR' }], total: 30, category_facets: [] })
    }
    return missing()
  })
}

const base = { apiBaseURL: 'http://backend/api/v1', publicApiBaseURL: 'https://api.test/api/v1' }

describe('a store in several languages', () => {
  it('serves each language under its own prefix: texts, prices, links and <html lang>', async () => {
    const out = await render({ ...base, url: '/store/k/en/products/red-shoes', fetch: backend() })
    expect(out.status).toBe(200)
    expect(out.lang).toBe('en')
    expect(out.html).toContain('€50.00')
    expect(out.html).toContain('No reviews yet.')
  })

  it('links pages in the language being read', async () => {
    const out = await render({ ...base, url: '/store/k/nl/', fetch: backend() })
    expect(out.lang).toBe('nl')
    expect(out.html).toContain('href="/store/k/nl/"')
  })

  it('plain URLs are the default language', async () => {
    const out = await render({ ...base, url: '/store/k/products/red-shoes', fetch: backend() })
    expect(out.lang).toBe('pt-BR')
    expect(out.html).toContain('Ainda não há avaliações.')
  })

  it('a language the store does not offer, or its default as a prefix, is not a page', async () => {
    expect((await render({ ...base, url: '/store/k/de/products/red-shoes', fetch: backend() })).status).toBe(404)
    expect((await render({ ...base, url: '/store/k/pt-BR/products/red-shoes', fetch: backend() })).status).toBe(404)
  })

  it('tells search engines about every language version', async () => {
    const out = await render({ ...base, url: '/store/k/en/products/red-shoes', publicOrigin: 'https://shop.test', fetch: backend() })
    expect(out.head).toContain('<link rel="canonical" href="https://shop.test/store/k/en/products/red-shoes">')
    expect(out.head).toContain('<link rel="alternate" hreflang="pt-BR" href="https://shop.test/store/k/products/red-shoes">')
    expect(out.head).toContain('<link rel="alternate" hreflang="en" href="https://shop.test/store/k/en/products/red-shoes">')
    expect(out.head).toContain('<link rel="alternate" hreflang="nl" href="https://shop.test/store/k/nl/products/red-shoes">')
    expect(out.head).toContain('<link rel="alternate" hreflang="x-default" href="https://shop.test/store/k/products/red-shoes">')
  })

  it('takes a returning shopper from a plain URL to the language they picked', async () => {
    const out = await render({ ...base, url: '/store/k/products/red-shoes?ref=mail', cookie: 'cos_shop_lang=en', fetch: backend() })
    expect(out.redirect).toBe('/store/k/en/products/red-shoes?ref=mail')
  })

  it('does not redirect for the default language, a prefixed URL or a language no longer offered', async () => {
    expect((await render({ ...base, url: '/store/k/products/red-shoes', cookie: 'cos_shop_lang=pt-BR', fetch: backend() })).redirect).toBeUndefined()
    expect((await render({ ...base, url: '/store/k/nl/products/red-shoes', cookie: 'cos_shop_lang=en', fetch: backend() })).redirect).toBeUndefined()
    expect((await render({ ...base, url: '/store/k/products/red-shoes', cookie: 'cos_shop_lang=de', fetch: backend() })).redirect).toBeUndefined()
  })

  it('lists every language version in the sitemap', async () => {
    const out = await renderSitemap({ storeKey: 'k', apiBaseURL: 'http://backend/api/v1', publicOrigin: 'https://shop.test', fetch: backend() })
    expect(out.body).toContain('<loc>https://shop.test/store/k/products/red-shoes</loc>')
    expect(out.body).toContain('<loc>https://shop.test/store/k/en/products/red-shoes</loc>')
    expect(out.body).toContain('<loc>https://shop.test/store/k/nl/</loc>')
  })
})

describe('translated merchant texts', () => {
  it('show in the language being read, and the default text where none was written', async () => {
    const home = docs.home
    docs.home = tree([{ id: 'hd', type: 'heading', props: { text: 'Bem-vindo', level: 1 }, translations: { en: { text: 'Welcome' } } }])
    try {
      expect((await render({ ...base, url: '/store/k/en/', fetch: backend() })).html).toContain('Welcome')
      expect((await render({ ...base, url: '/store/k/nl/', fetch: backend() })).html).toContain('Bem-vindo')
      expect((await render({ ...base, url: '/store/k/', fetch: backend() })).html).toContain('Bem-vindo')
    } finally {
      docs.home = home
    }
  })
})

describe('translated page texts', () => {
  it('title and description the page in the language being read', async () => {
    docMeta.home = { seo: { title: 'Loja', description: 'Produtos do Brasil' }, translations: { en: { seo: { title: 'Shop' } } } }
    try {
      const en = await render({ ...base, url: '/store/k/en/', fetch: backend() })
      expect(en.head).toContain('<title>Shop</title>')
      expect(en.head).toContain('content="Produtos do Brasil"')
      expect((await render({ ...base, url: '/store/k/nl/', fetch: backend() })).head).toContain('<title>Loja</title>')
    } finally {
      delete docMeta.home
    }
  })
})

describe('translated catalog texts', () => {
  it('asks the backend for products in the language being read, and hands that language to the browser', async () => {
    const f = backend()
    const out = await render({ ...base, url: '/store/k/en/products/red-shoes', fetch: f })
    const urls = f.mock.calls.map(c => String(c[0]))
    expect(urls).toContain('http://backend/api/v1/storefront/v1/products/by-slug/red-shoes?locale=en')
    expect(out.state).toContain('"locale":"en"')
  })

  it('asks for the default texts on plain URLs', async () => {
    const f = backend()
    await render({ ...base, url: '/store/k/products/red-shoes', fetch: f })
    expect(f.mock.calls.map(c => String(c[0]))).toContain('http://backend/api/v1/storefront/v1/products/by-slug/red-shoes')
  })
})
