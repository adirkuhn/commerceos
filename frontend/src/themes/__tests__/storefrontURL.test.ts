import { describe, it, expect } from 'vitest'
import { editorURL, storeURL, storefrontOrigin } from '../storefrontURL'

describe('storefront URLs', () => {
  it('prefers VITE_STOREFRONT_URL, else the API origin', () => {
    expect(storefrontOrigin({ VITE_STOREFRONT_URL: 'https://shop.example/', VITE_API_BASE_URL: 'https://api.example/api/v1' })).toBe('https://shop.example')
    expect(storefrontOrigin({ VITE_API_BASE_URL: 'https://api.example/api/v1' })).toBe('https://api.example')
    expect(storefrontOrigin({})).toBe('')
  })

  it('builds store and editor links', () => {
    expect(storeURL('chan-1', 'https://api.example')).toBe('https://api.example/store/chan-1/')
    expect(editorURL('a+b', 'https://api.example')).toBe('https://api.example/store/editor#session=a%2Bb')
  })

  it('passes the admin light/dark choice to the editor', () => {
    expect(editorURL('tok', 'https://api.example', 'dark')).toBe('https://api.example/store/editor#session=tok&theme=dark')
  })
})
