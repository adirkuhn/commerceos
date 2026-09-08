import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import * as authApi from '../api/auth'
import type { Account } from '../types'

const STORAGE_KEY = 'commerceos_session_token'

interface AuthContextValue {
  account: Account | null
  loading: boolean
  error: string | null
  // requestLogin emails a magic link -- always resolves (see
  // requestMagicLink's own doc comment), never reveals whether the email
  // was already a known customer.
  requestLogin: (email: string, redirectUrl: string) => Promise<void>
  // completeLogin exchanges a magic-link token (the "?token=" query param
  // on the redirect URL) for a session, called once by the login page's
  // verify step.
  completeLogin: (token: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [account, setAccount] = useState<Account | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = localStorage.getItem(STORAGE_KEY)
    if (!token) {
      setLoading(false)
      return
    }
    authApi.getAccount(token)
      .then(setAccount)
      .catch(() => localStorage.removeItem(STORAGE_KEY))
      .finally(() => setLoading(false))
  }, [])

  const requestLogin = useCallback(async (email: string, redirectUrl: string) => {
    setError(null)
    try {
      await authApi.requestMagicLink(email, redirectUrl)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    }
  }, [])

  const completeLogin = useCallback(async (token: string) => {
    setError(null)
    try {
      const session = await authApi.verifySession(token)
      localStorage.setItem(STORAGE_KEY, session.session_token)
      setAccount(session.account)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    }
  }, [])

  const logout = useCallback(async () => {
    const token = localStorage.getItem(STORAGE_KEY)
    if (!token) return
    setError(null)
    try {
      await authApi.logout(token)
    } finally {
      localStorage.removeItem(STORAGE_KEY)
      setAccount(null)
    }
  }, [])

  return (
    <AuthContext.Provider value={{ account, loading, error, requestLogin, completeLogin, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider')
  return ctx
}

export function getSessionToken(): string | null {
  return localStorage.getItem(STORAGE_KEY)
}
