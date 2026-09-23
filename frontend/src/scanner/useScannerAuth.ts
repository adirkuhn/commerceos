import { useState, useCallback, useEffect, useRef } from 'react'
import { msUntilRefresh, buildRefreshPayload } from '../auth/tokenRefresh'
import { APP_ID, lm2Origin } from '../auth/loginConfig'
import { refreshScannerSession, setOnTokenActivity } from './api'

export const SCANNER_TOKEN_KEY = 'commerceos_scanner_jwt'
export const SCANNER_REFRESH_KEY = 'commerceos_scanner_refresh'
const REFRESH_BUFFER_SECONDS = 60

// A pairing-code session's server-side TTL (claim_pairing_code.go's
// scannerSessionTTL) is a fixed 12h regardless of use. Rather than schedule
// a blind wall-clock renewal (which would need its own drift-prone timer),
// every successful API call reports its token here via api.ts's
// onTokenActivity hook -- once the token is more than half spent, that
// activity itself triggers a refresh. A picker who's actively scanning never
// runs the clock down; one left genuinely idle just rides out the original
// 12h and re-pairs, same as today.
const PAIRING_REFRESH_THRESHOLD_MS = 6 * 60 * 60 * 1000
// Refreshing on every single call once under the threshold would hammer the
// endpoint during a busy scan burst -- one attempt per cooldown window is
// plenty to keep a session that's still in use alive.
const PAIRING_REFRESH_COOLDOWN_MS = 10 * 60 * 1000

// Deliberately separate from ../auth/AuthContext's TOKEN_KEY -- a scanner
// device and the desktop admin session it was paired from are different
// browser storage entirely (often literally a different device). A
// pairing-code session (setToken) still has no refresh token -- see
// claim_pairing_code.go's scannerSessionTTL -- and needs none of the
// silent-renewal logic below. A Google-login session (setGoogleSession)
// does get one, straight from lm2's own app-login flow (same one
// ../auth/AuthContext uses), which is what makes it last lm2's configured
// refresh-token lifetime (days, not hours) instead of scannerSessionTTL's
// fixed 12h.
interface ScannerAuthState {
  token: string | null
  setToken: (token: string) => void
  setGoogleSession: (token: string, refreshToken: string) => void
  clearToken: () => void
}

function decodeExp(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1])) as { exp: number }
    return payload.exp
  } catch {
    return null
  }
}

function isExpired(token: string): boolean {
  const exp = decodeExp(token)
  return exp == null || exp * 1000 < Date.now()
}

// storeScannerSession is exported so ScannerCallbackPage can write a
// Google-login session's token+refresh token straight to the same storage
// this hook reads on next mount, without needing a live hook instance --
// the callback page redirects to /scanner right after, which remounts
// ScannerApp and picks the stored session up via loadStored below.
export function storeScannerSession(token: string, refreshToken: string): void {
  localStorage.setItem(SCANNER_TOKEN_KEY, token)
  localStorage.setItem(SCANNER_REFRESH_KEY, refreshToken)
}

function loadStored(): { token: string; refreshToken: string | null } | null {
  const raw = localStorage.getItem(SCANNER_TOKEN_KEY)
  if (!raw) return null
  if (isExpired(raw)) {
    localStorage.removeItem(SCANNER_TOKEN_KEY)
    localStorage.removeItem(SCANNER_REFRESH_KEY)
    return null
  }
  return { token: raw, refreshToken: localStorage.getItem(SCANNER_REFRESH_KEY) }
}

function refreshEndpoint(): string {
  return `${lm2Origin()}/api/v1/auth/app/refresh`
}

export function useScannerAuth(): ScannerAuthState {
  const [state, setState] = useState(loadStored)
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const stateRef = useRef(state)
  const lastPairingRefreshAttempt = useRef(0)

  useEffect(() => {
    stateRef.current = state
  }, [state])

  const setToken = useCallback((t: string) => {
    localStorage.setItem(SCANNER_TOKEN_KEY, t)
    localStorage.removeItem(SCANNER_REFRESH_KEY)
    setState({ token: t, refreshToken: null })
  }, [])

  const setGoogleSession = useCallback((t: string, refreshToken: string) => {
    storeScannerSession(t, refreshToken)
    setState({ token: t, refreshToken })
  }, [])

  const clearToken = useCallback(() => {
    localStorage.removeItem(SCANNER_TOKEN_KEY)
    localStorage.removeItem(SCANNER_REFRESH_KEY)
    if (refreshTimer.current) clearTimeout(refreshTimer.current)
    setState(null)
  }, [])

  // Activity-based renewal for pairing-code sessions (Google-login sessions
  // already get their own silent renewal below via refreshToken). Registers
  // once; reads live session state via stateRef so it never needs to be
  // re-registered as state changes.
  useEffect(() => {
    const handler = (usedToken: string) => {
      const current = stateRef.current
      if (!current || current.refreshToken) return
      if (current.token !== usedToken) return
      const exp = decodeExp(usedToken)
      if (exp == null) return
      const remainingMs = exp * 1000 - Date.now()
      if (remainingMs > PAIRING_REFRESH_THRESHOLD_MS) return
      const now = Date.now()
      if (now - lastPairingRefreshAttempt.current < PAIRING_REFRESH_COOLDOWN_MS) return
      lastPairingRefreshAttempt.current = now
      refreshScannerSession(usedToken)
        .then((res) => setToken(res.token))
        .catch(() => {
          // Best-effort: a failed refresh just leaves the existing token to
          // ride out its remaining TTL, same as before this feature existed.
        })
    }
    setOnTokenActivity(handler)
    return () => setOnTokenActivity(null)
  }, [setToken])

  // Silent renewal, same shape as ../auth/AuthContext's -- only armed for a
  // Google-login session (refreshToken set). A pairing-code session has none
  // and just rides out its fixed 12h window, same as before this existed.
  useEffect(() => {
    if (refreshTimer.current) clearTimeout(refreshTimer.current)
    if (!state?.refreshToken) return

    const exp = decodeExp(state.token)
    if (exp == null) return
    const delay = msUntilRefresh(exp, REFRESH_BUFFER_SECONDS, Date.now())
    refreshTimer.current = setTimeout(async () => {
      try {
        const res = await fetch(refreshEndpoint(), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(buildRefreshPayload(state.refreshToken!, APP_ID)),
        })
        if (!res.ok) throw new Error('refresh failed')
        const body = await res.json()
        setGoogleSession(body.data.access_token, body.data.refresh_token)
      } catch {
        clearToken()
      }
    }, delay)

    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state?.token])

  return { token: state?.token ?? null, setToken, setGoogleSession, clearToken }
}
