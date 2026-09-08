import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getStorefrontTheme } from '../theme'
import type { APIResponse, ThemePage } from '../../types'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

beforeEach(() => { vi.restoreAllMocks() })

const samplePage: ThemePage = {
  slug: 'home',
  canvas_width: 1440,
  blocks: [
    { id: 'b1', type: 'hero', layout: { desktop: { x: 0, y: 0, width: 1440, height: 400 }, mobile: null }, props: { heading: 'Welcome' } },
  ],
  custom_css: '',
}

describe('getStorefrontTheme', () => {
  it('returns the page document', async () => {
    mockFetch(samplePage)
    const result = await getStorefrontTheme('home')
    expect(result).toEqual(samplePage)
  })

  it('sends the storefront key as a bearer token against the right path', async () => {
    mockFetch(samplePage)
    await getStorefrontTheme('home')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/storefront/v1/theme/pages/home')
    expect((options?.headers as Record<string, string>).Authorization).toBe('Bearer test-storefront-key')
  })

  it('throws with the server error message on failure', async () => {
    mockFetchError('VALIDATION_ERROR', 'invalid page slug')
    await expect(getStorefrontTheme('home')).rejects.toThrow('invalid page slug')
  })
})
