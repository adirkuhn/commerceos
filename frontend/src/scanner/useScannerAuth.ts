import { useState, useCallback, useEffect, useRef } from 'react'
import { msUntilRefresh, buildRefreshPayload } from '../auth/tokenRefresh'
import { APP_ID, lm2Origin } from '../auth/loginConfig'

export const SCANNER_TOKEN_KEY = 'commerceos_scanner_jwt'
export const SCANNER_REFRESH_KEY = 'commerceos_scanner_refresh'
const REFRESH_BUFFER_SECONDS = 60

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
