import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { analytics } from '@/services/analytics'

interface WishlistState {
  productIds: string[]
  isWishlisted: (productId: string) => boolean
  add: (productId: string) => void
  remove: (productId: string) => void
  toggle: (productId: string) => void
}

export const useWishlistStore = create<WishlistState>()(
  persist(
    (set, get) => ({
      productIds: [],
      isWishlisted: (productId) => get().productIds.includes(productId),
      add: (productId) => {
        if (get().isWishlisted(productId)) return
        set(s => ({ productIds: [...s.productIds, productId] }))
        analytics.track('wishlist_added', { productId })
      },
      remove: (productId) => {
        set(s => ({ productIds: s.productIds.filter(id => id !== productId) }))
        analytics.track('wishlist_removed', { productId })
      },
      toggle: (productId) => (get().isWishlisted(productId) ? get().remove(productId) : get().add(productId)),
    }),
    { name: 'aurora-wishlist' },
  ),
)
