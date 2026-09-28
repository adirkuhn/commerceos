import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { Account, StorefrontAPI } from '../../api/storefront'
import { readToken, writeToken } from './storage'

export interface CustomerState {
  /** undefined until checked in the browser; null when signed out. */
  account: Account | null | undefined
  sessionToken: string | null
  /** Emails a sign-in link that returns to `returnURL`. */
  requestLink: (email: string, returnURL: string) => Promise<void>
  completeSignIn: (linkToken: string) => Promise<void>
  signOut: () => Promise<void>
}

const Ctx = createContext<CustomerState | null>(null)

/** The signed-in shopper (passwordless: emailed sign-in links). */
export function CustomerProvider({ api, storeKey, children }: { api: StorefrontAPI; storeKey: string; children: ReactNode }) {
  const [account, setAccount] = useState<Account | null | undefined>(undefined)
  const [sessionToken, setSessionToken] = useState<string | null>(null)

  useEffect(() => {
    const token = readToken('session', storeKey)
    if (!token) {
      setAccount(null)
      return
    }
    setSessionToken(token)
    api.getAccount(token).then(setAccount, () => {
      writeToken('session', storeKey, null)
      setSessionToken(null)
      setAccount(null)
    })
  }, [api, storeKey])

  const completeSignIn = useCallback(async (linkToken: string) => {
    const s = await api.verifySession(linkToken)
    writeToken('session', storeKey, s.session_token)
    setSessionToken(s.session_token)
    setAccount(s.account)
  }, [api, storeKey])

  const value = useMemo<CustomerState>(() => ({
    account, sessionToken, completeSignIn,
    requestLink: async (email, returnURL) => { await api.requestMagicLink(email, returnURL) },
    signOut: async () => {
      if (sessionToken) await api.logout(sessionToken).catch(() => undefined)
      writeToken('session', storeKey, null)
      setSessionToken(null)
      setAccount(null)
    },
  }), [account, api, completeSignIn, sessionToken, storeKey])

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useCustomer = () => useContext(Ctx)
