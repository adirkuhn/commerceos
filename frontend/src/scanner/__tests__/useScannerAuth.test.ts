// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useScannerAuth, storeScannerSession, SCANNER_TOKEN_KEY, SCANNER_REFRESH_KEY } from '../useScannerAuth'

function makeJWT(expSeconds: number): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ user_id: 'u1', tenant_id: 't1', role: 'customer_owner', exp: expSeconds }))
  return `${header}.${payload}.sig`
}

function mockFetchOnce(data: unknown, ok = true) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok, json: () => Promise.resolve({ data, error: ok ? null : { message: 'refresh failed' } }),
  }))
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
  vi.stubEnv('VITE_LM2_LOGIN_URL', 'https://lm2.example.com/login')
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useScannerAuth — pairing-code path (no refresh token)', () => {
  it('starts with no token when nothing stored', () => {
    const { result } = renderHook(() => useScannerAuth())
    expect(result.current.token).toBeNull()
  })

  it('clears an expired stored token on init', () => {
    const expired = makeJWT(Math.floor(Date.now() / 1000) - 3600)
    localStorage.setItem(SCANNER_TOKEN_KEY, expired)

    const { result } = renderHook(() => useScannerAuth())

    expect(result.current.token).toBeNull()
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBeNull()
  })

  it('setToken stores the token and clears any leftover refresh token', () => {
    localStorage.setItem(SCANNER_REFRESH_KEY, 'stale-refresh')
    const { result } = renderHook(() => useScannerAuth())
    const token = makeJWT(Math.floor(Date.now() / 1000) + 3600)

    act(() => result.current.setToken(token))

    expect(result.current.token).toBe(token)
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBe(token)
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBeNull()
  })

  it('never calls fetch for a pairing-code session, even well past its expiry', async () => {
    vi.useFakeTimers()
    const fetchSpy = vi.fn()
    vi.stubGlobal('fetch', fetchSpy)
    const { result } = renderHook(() => useScannerAuth())

    act(() => result.current.setToken(makeJWT(Math.floor(Date.now() / 1000) + 3600)))
    await vi.advanceTimersByTimeAsync(3600 * 1000 + 1000)

    expect(fetchSpy).not.toHaveBeenCalled()
  })
})

describe('useScannerAuth — Google-login path (refresh token)', () => {
  it('setGoogleSession stores both token and refresh token', () => {
    const { result } = renderHook(() => useScannerAuth())
    const token = makeJWT(Math.floor(Date.now() / 1000) + 3600)

    act(() => result.current.setGoogleSession(token, 'refresh-abc'))

    expect(result.current.token).toBe(token)
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBe(token)
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBe('refresh-abc')
  })

  it('silently refreshes shortly before the token expires', async () => {
    vi.useFakeTimers()
    const newToken = makeJWT(Math.floor(Date.now() / 1000) + 7200)
    mockFetchOnce({ access_token: newToken, refresh_token: 'refresh-2' })
    const { result } = renderHook(() => useScannerAuth())
    const token = makeJWT(Math.floor(Date.now() / 1000) + 3600)

    act(() => result.current.setGoogleSession(token, 'refresh-1'))
    // 60s buffer before the 1h expiry
    await act(async () => { await vi.advanceTimersByTimeAsync(3600 * 1000 - 60 * 1000) })

    expect(fetch).toHaveBeenCalledWith(
      'https://lm2.example.com/api/v1/auth/app/refresh',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ refresh_token: 'refresh-1', app: 'commerceos' }) }),
    )
    expect(result.current.token).toBe(newToken)
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBe('refresh-2')
  })

  it('clears the session when the refresh call fails', async () => {
    vi.useFakeTimers()
    mockFetchOnce(null, false)
    const { result } = renderHook(() => useScannerAuth())
    const token = makeJWT(Math.floor(Date.now() / 1000) + 3600)

    act(() => result.current.setGoogleSession(token, 'refresh-1'))
    await act(async () => { await vi.advanceTimersByTimeAsync(3600 * 1000 - 60 * 1000) })

    expect(result.current.token).toBeNull()
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBeNull()
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBeNull()
  })
})

describe('useScannerAuth — clearToken', () => {
  it('clears both keys and cancels any pending refresh', async () => {
    vi.useFakeTimers()
    const fetchSpy = vi.fn()
    vi.stubGlobal('fetch', fetchSpy)
    const { result } = renderHook(() => useScannerAuth())
    const token = makeJWT(Math.floor(Date.now() / 1000) + 3600)
    act(() => result.current.setGoogleSession(token, 'refresh-1'))

    act(() => result.current.clearToken())
    await vi.advanceTimersByTimeAsync(3600 * 1000 + 1000)

    expect(result.current.token).toBeNull()
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBeNull()
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBeNull()
    expect(fetchSpy).not.toHaveBeenCalled()
  })
})

describe('storeScannerSession', () => {
  it('writes both keys directly, for a caller with no live hook instance', () => {
    storeScannerSession('tok', 'ref')
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBe('tok')
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBe('ref')
  })
})
