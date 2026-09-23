export interface TokenIdentity {
  userId: string
  role: string
}

// Reads who a JWT belongs to, without verifying it -- display/UX decisions
// only (the backend enforces). Works for both the admin (lm2) token and the
// scanner's paired token, which carry the same user_id/role claims.
export function readClaims(token: string): TokenIdentity | null {
  const parts = token.split('.')
  if (parts.length !== 3) return null
  try {
    const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const claims = JSON.parse(atob(b64.padEnd(b64.length + ((4 - (b64.length % 4)) % 4), '='))) as { user_id?: unknown; role?: unknown }
    if (typeof claims.user_id !== 'string') return null
    return { userId: claims.user_id, role: typeof claims.role === 'string' ? claims.role : '' }
  } catch {
    return null
  }
}
