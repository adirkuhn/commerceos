import { describe, it, expect, vi } from 'vitest'
import { render } from '../entry-server'
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

const productDetail = {
  id: 'p1', name: 'Red shoes', slug: 'red-shoes', description: 'Comfy shoes for walking', category_id: null, tags: [], images: [],
  variants: [{ id: 'v1', name: 'Default', skus: [{ id: 's1', sku_code: 'RS', price_minor_units: 5000, currency_code: 'EUR', stock_quantity: 2 }] }],
}

function backend() {
  const ok = (data: unknown) => new Response(JSON.stringify({ data, error: null }))
  const missing = () => new Response(JSON.stringify({ data: null, error: { code: 'NOT_FOUND', message: 'x' } }), { status: 404 })
  return vi.fn<typeof fetch>(async input => {
    const url = new URL(String(input))
    const p = url.pathname.replace('/api/v1/storefront/v1', '')
    if (p === '/theme') return ok(theme)
    if (p === '/store') return ok({ name: 'Acme Goods', currency_code: 'EUR' })
    const doc = /^\/theme\/versions\/v1\/documents\/template\/(.+)$/.exec(p)
    if (doc) return ok({ kind: 'template', key: doc[1], title: '', seo: {}, root: docs[doc[1]] })
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

describe('SSR routes', () => {
  it('renders a product page by slug with its name as title and its data in the page', async () => {
    const out = await render({ ...base, url: '/store/k/products/red-shoes', fetch: backend() })
    expect(out.status).toBe(200)
    expect(out.html).toContain('Red shoes')
    expect(out.html).toContain('€ 50,00')
    expect(out.html).toContain('No reviews yet.')
    expect(out.head).toContain('<title>Red shoes</title>')
    expect(out.head).toContain('content="Comfy shoes for walking"')
  })

  it('404s a missing product with the not-found template', async () => {
    const out = await render({ ...base, url: '/store/k/products/nope', fetch: backend() })
    expect(out.status).toBe(404)
    expect(out.html).toContain('Not here')
  })

  it('renders a collection page with its listing, page size and SEO', async () => {
    const f = backend()
    const out = await render({ ...base, url: '/store/k/collections/summer?page=2', fetch: f })
    expect(out.status).toBe(200)
    expect(out.html).toContain('Summer')
    expect(out.html).toContain('Page 2 of 3')
    expect(out.head).toContain('<title>Summer sale</title>')
    const listCall = f.mock.calls.map(c => String(c[0])).find(u => u.includes('/storefront/v1/products?'))!
    expect(listCall).toContain('collection_id=col-1')
    expect(listCall).toContain('page=2')
    expect(listCall).toContain('per_page=12')
  })

  it('404s an unknown collection', async () => {
    const out = await render({ ...base, url: '/store/k/collections/nope', fetch: backend() })
    expect(out.status).toBe(404)
  })

  it('renders search results for the query', async () => {
    const f = backend()
    const out = await render({ ...base, url: '/store/k/search?q=red', fetch: f })
    expect(out.status).toBe(200)
    expect(out.html).toContain('Search results for “red”')
    expect(f.mock.calls.map(c => String(c[0])).some(u => u.includes('search=red'))).toBe(true)
  })

  it('shows the store name from the store info', async () => {
    const out = await render({ ...base, url: '/store/k/', fetch: backend() })
    expect(out.html).toContain('Acme Goods')
  })

  it('still renders when the store info is unavailable', async () => {
    const f = backend()
    const inner = f.getMockImplementation()!
    f.mockImplementation(async (input, init) => String(input).endsWith('/storefront/v1/store')
      ? new Response('{}', { status: 500 }) : inner(input, init))
    const out = await render({ ...base, url: '/store/k/', fetch: f })
    expect(out.status).toBe(200)
    expect(out.html).toContain('Store')
  })

  it('renders the order confirmation for the order in the URL', async () => {
    const out = await render({ ...base, url: '/store/k/orders/CO-1001', fetch: backend() })
    expect(out.status).toBe(200)
    expect(out.html).toContain('CO-1001')
    expect(out.html).toContain('Thanks!')
  })

  it('serves the newsletter pages itself, private and without acting on the token server-side', async () => {
    const f = backend()
    const out = await render({ ...base, url: '/store/k/newsletter/confirm?token=abc', fetch: f })
    expect(out.status).toBe(200)
    expect(out.html).toContain('Confirming your subscription')
    expect(out.head).toContain('<title>Newsletter</title>')
    expect(out.head).toContain('name="robots" content="noindex"')
    expect(out.head).toContain('name="referrer" content="no-referrer"')
    expect(f.mock.calls.some(c => String(c[0]).includes('/newsletter/'))).toBe(false)
  })

  it('marks up a product page for search engines and sharing', async () => {
    const out = await render({ ...base, publicOrigin: 'https://shop.test', url: '/store/k/products/red-shoes?utm_source=x', fetch: backend() })
    expect(out.head).toContain('<link rel="canonical" href="https://shop.test/store/k/products/red-shoes">')
    expect(out.head).toContain('<meta property="og:type" content="product">')
    expect(out.head).toContain('<meta property="og:site_name" content="Acme Goods">')
    const ld = [...out.head.matchAll(/<script type="application\/ld\+json">(.*?)<\/script>/g)].map(m => JSON.parse(m[1]))
    expect(ld.map(d => d['@type'])).toEqual(['Product', 'BreadcrumbList'])
    expect(ld[0].offers).toMatchObject({ '@type': 'Offer', price: '50.00', priceCurrency: 'EUR', availability: 'https://schema.org/InStock' })
    expect(ld[0]).not.toHaveProperty('aggregateRating')
    expect(out.head).not.toContain('noindex')
  })

  it('describes the store on its home page', async () => {
    const out = await render({ ...base, publicOrigin: 'https://shop.test', url: '/store/k/', fetch: backend() })
    expect(out.head).toContain('"@type":"Organization","name":"Acme Goods","url":"https://shop.test/store/k/"')
  })

  it('keeps shopper pages and search results out of search engines', async () => {
    for (const url of ['/store/k/search?q=red', '/store/k/orders/CO-1001']) {
      const out = await render({ ...base, publicOrigin: 'https://shop.test', url, fetch: backend() })
      expect(out.head, url).toContain('<meta name="robots" content="noindex">')
      expect(out.head, url).not.toContain('rel="canonical"')
      expect(out.head, url).not.toContain('ld+json')
    }
  })

  it('leaves canonical URLs out when no public origin is configured', async () => {
    const out = await render({ ...base, url: '/store/k/products/red-shoes', fetch: backend() })
    expect(out.head).not.toContain('rel="canonical"')
    expect(out.head).toContain('"@type":"Product"')
  })
})
