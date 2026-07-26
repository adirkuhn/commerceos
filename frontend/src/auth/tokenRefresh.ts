// exp is a JWT-style NumericDate (seconds since epoch), matching how
// AuthContext already decodes claims.
export function msUntilRefresh(expSeconds: number, bufferSeconds: number, nowMs: number): number {
  const delay = expSeconds * 1000 - bufferSeconds * 1000 - nowMs
  return Math.max(delay, 0)
}

export function buildRefreshPayload(refreshToken: string, app: string): { refresh_token: string; app: string } {
  return { refresh_token: refreshToken, app }
}
