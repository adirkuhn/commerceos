import { createContext, useContext, type ReactNode } from 'react'
import type { StorefrontAPI } from '../api/storefront'

const ApiContext = createContext<StorefrontAPI | null>(null)

export function ApiProvider({ api, children }: { api: StorefrontAPI; children: ReactNode }) {
  return <ApiContext.Provider value={api}>{children}</ApiContext.Provider>
}

export function useStorefrontAPI(): StorefrontAPI {
  const api = useContext(ApiContext)
  if (!api) throw new Error('useStorefrontAPI outside ApiProvider')
  return api
}
