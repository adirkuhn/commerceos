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

  it('turns error envelopes into ApiError with status and code', async () => {
    const f = fakeFetch({ data: null, error: { code: 'NOT_FOUND', message: 'no live theme' } }, 404)
    const err = await new StorefrontAPI({ baseURL: 'https://api.test/api/v1', storeKey: 'k', fetch: f }).getTheme().catch(e => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 404, code: 'NOT_FOUND', message: 'no live theme' })
  })
})
