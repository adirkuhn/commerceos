import { useState, useCallback } from 'react'

const TOKEN_KEY = 'commerceos_scanner_jwt'

// Deliberately separate from ../auth/AuthContext's TOKEN_KEY -- a scanner
// device and the desktop admin session it was paired from are different
// browser storage entirely (often literally a different device), and the
// scanner's token has no refresh token (see claim_pairing_code.go's
// scannerSessionTTL) so it needs none of AuthContext's silent-renewal logic.
interface ScannerAuthState {
  token: string | null
  setToken: (token: string) => void
  clearToken: () => void
}

function isExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1])) as { exp: number }
    return payload.exp * 1000 < Date.now()
  } catch {
    return true
  }
}

function loadStored(): string | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  if (isExpired(raw)) {
    localStorage.removeItem(TOKEN_KEY)
    return null
  }
  return raw
}

export function useScannerAuth(): ScannerAuthState {
  const [token, setTokenState] = useState<string | null>(loadStored)

  const setToken = useCallback((t: string) => {
    localStorage.setItem(TOKEN_KEY, t)
    setTokenState(t)
  }, [])

  const clearToken = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    setTokenState(null)
  }, [])

  return { token, setToken, clearToken }
}
