// hasLoginUrl guards LoginPage: if lm2 isn't configured, show an
// admin-facing message instead of a login button that would 404.
export function hasLoginUrl(env: { VITE_LM2_LOGIN_URL?: string }): boolean {
  return Boolean(env.VITE_LM2_LOGIN_URL)
}

// APP_ID identifies this app to lm2's app-login allowlist (APP_REDIRECT_ALLOWLIST).
export const APP_ID = 'commerceos'
