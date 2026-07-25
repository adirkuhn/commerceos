import { describe, it, expect } from 'vitest'

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
