import { create } from 'zustand'
import { persist } from 'zustand/middleware'

const MAX_RECENT_SEARCHES = 6

interface RecentSearchesState {
  queries: string[]
  record: (query: string) => void
  clear: () => void
}

export const useRecentSearchesStore = create<RecentSearchesState>()(
  persist(
    (set, get) => ({
      queries: [],
      record: (query) => {
        const trimmed = query.trim()
        if (!trimmed) return
        const rest = get().queries.filter(q => q.toLowerCase() !== trimmed.toLowerCase())
        set({ queries: [trimmed, ...rest].slice(0, MAX_RECENT_SEARCHES) })
      },
      clear: () => set({ queries: [] }),
    }),
    { name: 'aurora-recent-searches' },
  ),
)
