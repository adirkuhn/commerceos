import { describe, it, expect, vi, beforeEach } from 'vitest'
import { request, setStorefrontKey } from '../client'

function mockFetchOk() {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve({ data: {}, error: null }) }))
}

beforeEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

describe('setStorefrontKey', () => {
  it('makes every subsequent request use the new key as its bearer token', async () => {
    mockFetchOk()
    setStorefrontKey('channel-key-1')

    await request('/products')

    const [, options] = vi.mocked(fetch).mock.calls[0]
    expect((options?.headers as Record<string, string>).Authorization).toBe('Bearer channel-key-1')
  })

  it('persists the key across a fresh module load (a new tab/visit), via localStorage', () => {
    setStorefrontKey('channel-key-2')

    expect(localStorage.getItem('commerceos_storefront_key')).toBe('channel-key-2')
  })
})
