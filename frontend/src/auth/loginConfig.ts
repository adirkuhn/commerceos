// hasLoginUrl guards LoginPage: if lm2 isn't configured, show an
// admin-facing message instead of a login button that would 404.
export function hasLoginUrl(env: { VITE_LM2_LOGIN_URL?: string }): boolean {
  return Boolean(env.VITE_LM2_LOGIN_URL)
}

// APP_ID identifies this app to lm2's app-login allowlist (APP_REDIRECT_ALLOWLIST).
export const APP_ID = 'commerceos'

// lm2Origin is lm2's own API origin, derived from the login URL -- used by
// any direct frontend-to-lm2 call (token refresh, company member/invite
// management) the same way AuthContext already derives its refresh endpoint.
export function lm2Origin(): string {
  return new URL(import.meta.env.VITE_LM2_LOGIN_URL as string).origin
}
