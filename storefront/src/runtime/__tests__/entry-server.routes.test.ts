import { describe, it, expect, vi } from 'vitest'
import { render } from '../entry-server'
import { defaultSettings } from '../../core/settings'

const theme = {
  version_id: 'v1', version_number: 1, preview: false, settings: defaultSettings(),
  documents: ['home', 'product', 'collection', 'search', 'not-found'].map(key => ({ kind: 'template', key, title: '' })),
}

const tree = (children: unknown[]) => ({ id: 'root', type: 'root', children: [{ id: 's', type: 'section', children }] })
const docs: Record<string, unknown> = {
  product: tree([{ id: 't', type: 'product-title', props: { level: 1 } }, { id: 'pr', type: 'product-price', props: {} },
    { id: 'rv', type: 'product-reviews', props: { title: 'Reviews', limit: 3 } }]),
  collection: tree([{ id: 'h', type: 'collection-header', props: {} }, { id: 'p', type: 'collection-products', props: { per_page: 12 } }, { id: 'pg', type: 'pagination', props: {} }]),
  search: tree([{ id: 'h', type: 'collection-header', props: {} }, { id: 'p', type: 'collection-products', props: {} }]),
  'not-found': tree([{ id: 'nf', type: 'heading', props: { text: 'Not here', level: 1 } }]),
  home: tree([]),
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
})
