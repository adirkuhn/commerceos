import { create } from 'zustand'
import { persist } from 'zustand/middleware'

const MAX_COMPARE_ITEMS = 4

interface CompareState {
  productIds: string[]
  isComparing: (productId: string) => boolean
  add: (productId: string) => void
  remove: (productId: string) => void
  toggle: (productId: string) => void
  clear: () => void
}

export const useCompareStore = create<CompareState>()(
  persist(
    (set, get) => ({
      productIds: [],
      isComparing: (productId) => get().productIds.includes(productId),
      add: (productId) => {
        if (get().isComparing(productId) || get().productIds.length >= MAX_COMPARE_ITEMS) return
        set(s => ({ productIds: [...s.productIds, productId] }))
      },
      remove: (productId) => set(s => ({ productIds: s.productIds.filter(id => id !== productId) })),
      toggle: (productId) => (get().isComparing(productId) ? get().remove(productId) : get().add(productId)),
      clear: () => set({ productIds: [] }),
    }),
    { name: 'aurora-compare' },
  ),
)

export { MAX_COMPARE_ITEMS }
