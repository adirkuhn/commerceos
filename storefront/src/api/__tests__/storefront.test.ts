import { describe, it, expect, vi } from 'vitest'
import { ApiError, StorefrontAPI } from '../storefront'
import type { StorefrontTheme } from '../../core/document'
import { defaultSettings } from '../../core/settings'

const theme: StorefrontTheme = { version_id: 'v1', version_number: 1, preview: false, settings: defaultSettings(), documents: [] }

function fakeFetch(body: unknown, status = 200) {
  return vi.fn<typeof fetch>(async () => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }))
}

describe('StorefrontAPI', () => {
  it('sends the store key as bearer and unwraps the envelope', async () => {
    const f = fakeFetch({ data: theme, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'key-1', fetch: f })

    await expect(api.getTheme()).resolves.toEqual(theme)
    const [url, init] = f.mock.calls[0]
    expect(url).toBe('https://api.test/api/v1/storefront/v1/theme')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer key-1')
    expect(new Headers(init?.headers).has('X-Theme-Preview')).toBe(false)
  })

  it('requests a sign-in link with only the email: the server decides where it points', async () => {
    const f = fakeFetch({ data: null, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f })
    await api.requestMagicLink('ann@x.test')
    expect(JSON.parse(String(f.mock.calls[0][1]?.body))).toEqual({ email: 'ann@x.test' })
  })

  it('reads published documents by version', async () => {
    const f = fakeFetch({ data: { kind: 'template', key: 'home' }, error: null })
    await new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f }).getDocument(theme, 'template', 'home')
    expect(f.mock.calls[0][0]).toBe('https://api.test/api/v1/storefront/v1/theme/versions/v1/documents/template/home')
  })

  it('reads the draft through the preview endpoint when previewing', async () => {
    const f = fakeFetch({ data: { kind: 'template', key: 'home' }, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', previewToken: 'pv', fetch: f })
    await api.getDocument({ ...theme, preview: true }, 'page', 'about us')
    expect(f.mock.calls[0][0]).toBe('https://api.test/api/v1/storefront/v1/theme/preview/documents/page/about%20us')
    expect(new Headers(f.mock.calls[0][1]?.headers).get('X-Theme-Preview')).toBe('pv')
  })

  it('lists products by category or explicit ids', async () => {
    const f = fakeFetch({ data: { items: [], total: 0 }, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f })
    await api.listProducts({ categoryId: 'c1', perPage: 8 })
    await api.listProducts({ productIds: ['a', 'b'] })
    expect(f.mock.calls[0][0]).toBe('https://api.test/api/v1/storefront/v1/products?category_id=c1&per_page=8')
    expect(f.mock.calls[1][0]).toBe('https://api.test/api/v1/storefront/v1/products?product_ids=a%2Cb&per_page=12')
  })

  it('asks for catalog texts in the shopper\u2019s language when it isn\u2019t the default', async () => {
    const f = fakeFetch({ data: { items: [], total: 0 }, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', locale: 'pt-BR', fetch: f })
    await api.listProducts({ search: 'pão' })
    await api.getProductBySlug('red shoes')
    await api.getRecommendations('p1')
    await api.listCollections()
    await api.getCollection('summer')
    await api.getStore()
    expect(f.mock.calls.map(c => String(c[0]).replace('https://api.test/api/v1/storefront/v1', ''))).toEqual([
      '/products?search=p%C3%A3o&per_page=12&locale=pt-BR',
      '/products/by-slug/red%20shoes?locale=pt-BR',
      '/products/p1/recommendations?locale=pt-BR',
      '/collections?per_page=100&locale=pt-BR',
      '/collections/by-slug/summer?locale=pt-BR',
      '/store',
    ])
  })

  it('turns error envelopes into ApiError with status and code', async () => {
    const f = fakeFetch({ data: null, error: { code: 'NOT_FOUND', message: 'no live theme' } }, 404)
    const err = await new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f }).getTheme().catch(e => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 404, code: 'NOT_FOUND', message: 'no live theme' })
  })
  it('reads products by slug, collections and store info', async () => {
    const f = fakeFetch({ data: {}, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f })
    await api.getProductBySlug('red shoes')
    await api.getCollection('summer')
    await api.getStore()
    await api.listProducts({ collectionId: 'c1', page: 2, perPage: 24 })
    expect(f.mock.calls.map(c => c[0])).toEqual([
      'https://api.test/api/v1/storefront/v1/products/by-slug/red%20shoes',
      'https://api.test/api/v1/storefront/v1/collections/by-slug/summer',
      'https://api.test/api/v1/storefront/v1/store',
      'https://api.test/api/v1/storefront/v1/products?collection_id=c1&page=2&per_page=24',
    ])
  })

  it('manages carts with whole-line-set updates', async () => {
    const f = fakeFetch({ data: { token: 't', items: [] }, error: null })
    const api = new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f })
    await api.createCart()
    await api.setCartItems('t', [{ sku_code: 'A', quantity: 2 }])
    expect(f.mock.calls[0][1]?.method).toBe('POST')
    expect(f.mock.calls[1][0]).toBe('https://api.test/api/v1/storefront/v1/carts/t/items')
    expect(f.mock.calls[1][1]?.method).toBe('PUT')
    expect(JSON.parse(String(f.mock.calls[1][1]?.body))).toEqual({ items: [{ sku_code: 'A', quantity: 2 }] })
  })
})
