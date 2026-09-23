import { describe, it, expect } from 'vitest'
import { readClaims } from '../jwtClaims'

function token(payload: object): string {
  const b64 = (o: object) => btoa(JSON.stringify(o)).replace(/=+$/, '').replace(/\+/g, '-').replace(/\//g, '_')
  return `${b64({ alg: 'HS256' })}.${b64(payload)}.sig`
}

describe('readClaims', () => {
  it('reads user_id and role from a JWT payload', () => {
    expect(readClaims(token({ user_id: 'u1', role: 'customer_owner', tenant_id: 't' }))).toEqual({ userId: 'u1', role: 'customer_owner' })
  })

  it('handles base64url payloads', () => {
    // "~~~" encodes to characters that differ between base64 and base64url
    expect(readClaims(token({ user_id: '~~~?', role: 'customer_user' }))?.userId).toBe('~~~?')
  })

  it('returns null for garbage', () => {
    expect(readClaims('')).toBeNull()
    expect(readClaims('a.b')).toBeNull()
    expect(readClaims('a.!!!.c')).toBeNull()
  })
})
