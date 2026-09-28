import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { StorefrontAPI } from '../../api/storefront'
import { readToken, writeToken } from './storage'

export interface WishlistState {
  productIds: string[] | undefined
  busy: boolean
  toggle: (productId: string) => Promise<void>
}

const Ctx = createContext<WishlistState | null>(null)

/** The shopper's saved products, kept by token in this browser. */
export function WishlistProvider({ api, storeKey, children }: { api: StorefrontAPI; storeKey: string; children: ReactNode }) {
  const [token, setToken] = useState<string | null>(null)
  const [productIds, setProductIds] = useState<string[] | undefined>(undefined)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    const t = readToken('wishlist', storeKey)
    if (!t) {
      setProductIds([])
      return
    }
    setToken(t)
    api.getWishlist(t).then(w => setProductIds(w.product_ids), () => {
      writeToken('wishlist', storeKey, null)
      setToken(null)
      setProductIds([])
    })
  }, [api, storeKey])

  const value = useMemo<WishlistState>(() => ({
    productIds, busy,
    toggle: async productId => {
      setBusy(true)
      try {
        let t = token
        if (!t) {
          t = (await api.createWishlist()).token
          writeToken('wishlist', storeKey, t)
          setToken(t)
        }
        setProductIds((await api.toggleWishlistItem(t, productId)).product_ids)
      } finally {
        setBusy(false)
      }
    },
  }), [api, busy, productIds, storeKey, token])

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useWishlist = () => useContext(Ctx)
