import { afterEach, describe, expect, it, vi } from 'vitest'
import { appEnv } from '../appEnv'

describe('appEnv', () => {
  afterEach(() => {
    delete globalThis.__COMMERCEOS_CONFIG__
    vi.unstubAllEnvs()
  })

  it('uses the build-time values when the server wrote no config (dev, tests)', () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://build.example/api/v1')

    expect(appEnv().VITE_API_BASE_URL).toBe('https://build.example/api/v1')
  })

  it('prefers the config the server wrote at startup', () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://build.example/api/v1')
    globalThis.__COMMERCEOS_CONFIG__ = { VITE_API_BASE_URL: 'https://server.example/api/v1' }

    expect(appEnv().VITE_API_BASE_URL).toBe('https://server.example/api/v1')
  })

  it('falls back per setting: a blank or missing server value keeps the build-time one', () => {
    vi.stubEnv('VITE_LM2_LOGIN_URL', 'https://build.example/app-login')
    vi.stubEnv('VITE_LM2_CALLBACK_PATH', '/auth/callback')
    globalThis.__COMMERCEOS_CONFIG__ = { VITE_LM2_LOGIN_URL: '' }

    expect(appEnv().VITE_LM2_LOGIN_URL).toBe('https://build.example/app-login')
    expect(appEnv().VITE_LM2_CALLBACK_PATH).toBe('/auth/callback')
  })

  it('reads the values at call time, not once at import', () => {
    expect(appEnv().VITE_STOREFRONT_URL).toBeFalsy()
    globalThis.__COMMERCEOS_CONFIG__ = { VITE_STOREFRONT_URL: 'https://shop.example' }

    expect(appEnv().VITE_STOREFRONT_URL).toBe('https://shop.example')
  })
})
