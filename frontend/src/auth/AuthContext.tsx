import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

const TOKEN_KEY = 'commerceos_jwt'

export interface JWTClaims {
  sub: string
  email: string
  company_id: string
  pos_slug: string
  jti: string
  type: string
  exp: number
  iat: number
  license_expires_at: number
}

interface AuthContextValue {
  token: string | null
  user: JWTClaims | null
  login: (token: string) => void
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

function loadStored(): { token: string; user: JWTClaims } | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  const user = decodeToken(raw)
  if (!user) {
    localStorage.removeItem(TOKEN_KEY)
    return null
  }
  return { token: raw, user }
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ token: string; user: JWTClaims } | null>(loadStored)

  const login = useCallback((raw: string) => {
    const user = decodeToken(raw)
    if (!user) return
    localStorage.setItem(TOKEN_KEY, raw)
    setState({ token: raw, user })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    setState(null)
  }, [])

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
