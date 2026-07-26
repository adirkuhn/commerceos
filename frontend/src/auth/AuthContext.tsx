import { createContext, useContext, useState, useCallback, useEffect, useRef, type ReactNode } from 'react'
import { msUntilRefresh, buildRefreshPayload } from './tokenRefresh'
import { APP_ID } from './loginConfig'

const TOKEN_KEY = 'commerceos_jwt'
const REFRESH_KEY = 'commerceos_refresh'
const REFRESH_BUFFER_SECONDS = 60

// Shape lm2's app-login flow issues — matches backend/infrastructure/middleware/auth.go's Claims exactly.
export interface JWTClaims {
  user_id: string
  tenant_id: string
  role: string
  exp: number
  iat: number
  sub?: string
}

interface AuthContextValue {
  token: string | null
  user: JWTClaims | null
  login: (token: string, refreshToken: string) => void
  logout: () => void
  isAuthenticated: boolean
}

function decodeToken(raw: string): JWTClaims | null {
  try {
    const parts = raw.split('.')
    if (parts.length !== 3) return null
    const claims = JSON.parse(atob(parts[1])) as JWTClaims
    if (claims.exp * 1000 < Date.now()) return null
    return claims
  } catch {
    return null
  }
}

function loadStored(): { token: string; refreshToken: string | null; user: JWTClaims } | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  const user = decodeToken(raw)
  if (!user) {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_KEY)
    return null
  }
  return { token: raw, refreshToken: localStorage.getItem(REFRESH_KEY), user }
}

function refreshEndpoint(): string {
  return `${new URL(import.meta.env.VITE_LM2_LOGIN_URL as string).origin}/api/v1/auth/app/refresh`
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ token: string; refreshToken: string | null; user: JWTClaims } | null>(loadStored)
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const login = useCallback((raw: string, refreshToken: string) => {
    const user = decodeToken(raw)
    if (!user) return
    localStorage.setItem(TOKEN_KEY, raw)
    localStorage.setItem(REFRESH_KEY, refreshToken)
    setState({ token: raw, refreshToken, user })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_KEY)
    if (refreshTimer.current) clearTimeout(refreshTimer.current)
    setState(null)
  }, [])

  // Silent renewal: refresh the access token shortly before it expires so
  // the session survives past its TTL without bouncing back through lm2.
  useEffect(() => {
    if (refreshTimer.current) clearTimeout(refreshTimer.current)
    if (!state?.refreshToken) return

    const delay = msUntilRefresh(state.user.exp, REFRESH_BUFFER_SECONDS, Date.now())
    refreshTimer.current = setTimeout(async () => {
      try {
        const res = await fetch(refreshEndpoint(), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(buildRefreshPayload(state.refreshToken!, APP_ID)),
        })
        if (!res.ok) throw new Error('refresh failed')
        const body = await res.json()
        login(body.data.access_token, body.data.refresh_token)
      } catch {
        logout()
      }
    }, delay)

    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state?.token])

  return (
    <AuthContext.Provider value={{
      token: state?.token ?? null,
      user: state?.user ?? null,
      login,
      logout,
      isAuthenticated: !!state,
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
