import { create } from 'zustand'
import { persist } from 'zustand/middleware'

const MAX_RECENT = 12

interface RecentlyViewedState {
  productIds: string[]
  record: (productId: string) => void
}

export const useRecentlyViewedStore = create<RecentlyViewedState>()(
  persist(
    (set, get) => ({
      productIds: [],
      record: (productId) => {
        const rest = get().productIds.filter(id => id !== productId)
        set({ productIds: [productId, ...rest].slice(0, MAX_RECENT) })
      },
    }),
    { name: 'aurora-recently-viewed' },
  ),
)
