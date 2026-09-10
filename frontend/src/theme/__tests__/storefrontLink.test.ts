import { describe, expect, it, vi } from 'vitest'
import { storefrontEditURL, storefrontEnterPath, storefrontOrigin, storefrontPreviewURL, storefrontVisitURL } from '../storefrontLink'

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

describe('storefrontPreviewURL', () => {
  it('maps a slug with a fixed route to its real storefront page', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontPreviewURL('chan-1', 'cart')).toBe('https://shop.example.com/store/enter/chan-1?next=%2Fcart')
  })

  it('falls back "product" to the products list -- no specific product id to preview generically', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontPreviewURL('chan-1', 'product')).toBe('https://shop.example.com/store/enter/chan-1?next=%2Fproducts')
  })

  it('returns "" for a slug with no previewable route (order-confirmation needs a real order)', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontPreviewURL('chan-1', 'order-confirmation')).toBe('')
  })
})

describe('storefrontEditURL', () => {
  it('builds the embedded Puck editor URL under the /store path prefix, with the session token encoded', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontEditURL('home', 'edit-token-1')).toBe('https://shop.example.com/store/edit/home?token=edit-token-1')
  })

  it('encodes special characters in the token', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.example.com')
    expect(storefrontEditURL('home', 'a+b/c')).toBe('https://shop.example.com/store/edit/home?token=a%2Bb%2Fc')
  })

  it('returns "" when the origin can\'t be computed', () => {
    vi.stubEnv('VITE_STOREFRONT_URL', '')
    vi.stubEnv('VITE_API_BASE_URL', 'not-a-url')
    expect(storefrontEditURL('home', 'edit-token-1')).toBe('')
  })
})
