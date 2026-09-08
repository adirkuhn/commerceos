import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import * as wishlistApi from '../api/wishlist'
import type { Wishlist } from '../types'

const STORAGE_KEY = 'commerceos_wishlist_token'

interface WishlistContextValue {
  wishlist: Wishlist | null
  loading: boolean
  error: string | null
  isSaved: (productId: string) => boolean
  toggle: (productId: string) => Promise<void>
}

const WishlistContext = createContext<WishlistContextValue | null>(null)

export function WishlistProvider({ children }: { children: ReactNode }) {
  const [wishlist, setWishlist] = useState<Wishlist | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = localStorage.getItem(STORAGE_KEY)
    if (!token) {
      setLoading(false)
      return
    }
    wishlistApi.getWishlist(token)
      .then(setWishlist)
      .catch(() => localStorage.removeItem(STORAGE_KEY))
      .finally(() => setLoading(false))
  }, [])

  // Same "create a token lazily on first use" pattern as CartContext's
  // ensureCart -- a shopper who never saves anything never gets a
  // wishlist row at all.
  const ensureWishlist = useCallback(async (): Promise<Wishlist> => {
    if (wishlist) return wishlist
    const existingToken = localStorage.getItem(STORAGE_KEY)
    if (existingToken) {
      try {
        const existing = await wishlistApi.getWishlist(existingToken)
        setWishlist(existing)
        return existing
      } catch {
        localStorage.removeItem(STORAGE_KEY)
      }
    }
    const created = await wishlistApi.createWishlist()
    localStorage.setItem(STORAGE_KEY, created.token)
    setWishlist(created)
    return created
  }, [wishlist])

  const isSaved = useCallback((productId: string) => wishlist?.product_ids.includes(productId) ?? false, [wishlist])

  const toggle = useCallback(async (productId: string) => {
    setError(null)
    try {
      const current = await ensureWishlist()
      const updated = await wishlistApi.toggleWishlistItem(current.token, productId)
      setWishlist(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    }
  }, [ensureWishlist])

  return (
    <WishlistContext.Provider value={{ wishlist, loading, error, isSaved, toggle }}>
      {children}
    </WishlistContext.Provider>
  )
}

export function useWishlist(): WishlistContextValue {
  const ctx = useContext(WishlistContext)
  if (!ctx) throw new Error('useWishlist must be used within a WishlistProvider')
  return ctx
}
