import { describe, it, expect } from 'vitest'
import { msUntilRefresh, buildRefreshPayload } from '../tokenRefresh'
import { hasLoginUrl } from '../loginConfig'

// JWT utilities used by AuthContext — test the logic in isolation
function isExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    if (payload.exp && payload.exp * 1000 < Date.now()) return true
    return false
  } catch {
    return true
  }
}

function isValidStructure(token: string): boolean {
  return token.split('.').length === 3
}

function makeJWT(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fakesig`
}

describe('JWT expiry check', () => {
  it('returns false for non-expired token', () => {
    const future = Math.floor(Date.now() / 1000) + 3600
    const token = makeJWT({ exp: future, user_id: 'u1' })
    expect(isExpired(token)).toBe(false)
  })

  it('returns true for expired token', () => {
    const past = Math.floor(Date.now() / 1000) - 3600
    const token = makeJWT({ exp: past, user_id: 'u1' })
    expect(isExpired(token)).toBe(true)
  })

  it('returns true for malformed token', () => {
    expect(isExpired('not.a')).toBe(true)
    expect(isExpired('garbage')).toBe(true)
  })

  it('returns false for token without exp claim', () => {
    const token = makeJWT({ user_id: 'u1' })
    expect(isExpired(token)).toBe(false)
  })
})

describe('JWT structure validation', () => {
  it('accepts valid 3-part JWT', () => {
    const token = makeJWT({ user_id: 'u1' })
    expect(isValidStructure(token)).toBe(true)
  })

  it('rejects 2-part strings', () => {
    expect(isValidStructure('a.b')).toBe(false)
  })

  it('rejects non-JWT strings', () => {
    expect(isValidStructure('plaintext')).toBe(false)
  })
})

describe('callback URL extraction', () => {
  it('extracts token from query string', () => {
    const token = makeJWT({ user_id: 'u1' })
    const params = new URLSearchParams(`?token=${token}`)
    expect(params.get('token')).toBe(token)
  })

  it('returns null when token absent', () => {
    const params = new URLSearchParams('?code=abc')
    expect(params.get('token')).toBeNull()
  })
})

describe('msUntilRefresh', () => {
  it('computes delay to buffer seconds before expiry', () => {
    const nowMs = 1_000_000
    const expSeconds = nowMs / 1000 + 3600 // exp is 1 hour ahead, in seconds
    const bufferSeconds = 60
    const got = msUntilRefresh(expSeconds, bufferSeconds, nowMs)
    expect(got).toBe(3600 * 1000 - bufferSeconds * 1000)
  })

  it('clamps to 0 when already past the refresh point', () => {
    const nowMs = 1_000_000
    const expSeconds = nowMs / 1000 + 30 // only 30s ahead
    const bufferSeconds = 60 // buffer alone exceeds time left
    expect(msUntilRefresh(expSeconds, bufferSeconds, nowMs)).toBe(0)
  })

  it('clamps to 0 for an already-expired token', () => {
    const nowMs = 1_000_000
    const expSeconds = nowMs / 1000 - 3600
    expect(msUntilRefresh(expSeconds, 60, nowMs)).toBe(0)
  })
})

describe('buildRefreshPayload', () => {
  it('shapes the payload lm2 expects', () => {
    expect(buildRefreshPayload('rt-123', 'commerceos')).toEqual({
      refresh_token: 'rt-123',
      app: 'commerceos',
    })
  })
})

describe('hasLoginUrl', () => {
  it('returns true when VITE_LM2_LOGIN_URL is set', () => {
    expect(hasLoginUrl({ VITE_LM2_LOGIN_URL: 'https://lm2.dev.localhost/app-login' })).toBe(true)
  })

  it('returns false when unset', () => {
    expect(hasLoginUrl({})).toBe(false)
  })

  it('returns false when empty string', () => {
    expect(hasLoginUrl({ VITE_LM2_LOGIN_URL: '' })).toBe(false)
  })
})
