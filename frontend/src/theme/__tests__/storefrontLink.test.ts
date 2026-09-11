import { describe, expect, it, vi } from 'vitest'
import { storefrontEnterPath, storefrontOrigin, storefrontVisitURL } from '../storefrontLink'

describe('storefrontOrigin', () => {
  it('prefers VITE_STOREFRONT_URL when set', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontOrigin()).toBe('https://shop.example.com')
  })

  it('falls back to VITE_API_BASE_URL\'s own origin -- a working default with no extra configuration', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', '')
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/api/v1')
    expect(storefrontOrigin()).toBe('https://api.example.com')
  })

  it('returns "" when neither is usable', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', '')
    vi.stubEnv('VITE_API_BASE_URL', 'not-a-url')
    expect(storefrontOrigin()).toBe('')
  })
})

describe('storefrontEnterPath', () => {
  it('builds the bare entry path with no next', () => {
    expect(storefrontEnterPath('chan-1')).toBe('/store/enter/chan-1')
  })

  it('appends an encoded next when given', () => {
    expect(storefrontEnterPath('chan-1', '/products')).toBe('/store/enter/chan-1?next=%2Fproducts')
  })
})

describe('storefrontVisitURL', () => {
  it('combines the origin and the bare entry path', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontVisitURL('chan-1')).toBe('https://shop.example.com/store/enter/chan-1')
  })

  it('returns "" when the origin can\'t be computed', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', '')
    vi.stubEnv('VITE_API_BASE_URL', 'not-a-url')
    expect(storefrontVisitURL('chan-1')).toBe('')
  })
})
