import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, X, Clock, TrendingUp } from 'lucide-react'
import { Input } from '@/components/ui/Input'
import { useDebounce } from '@/hooks/useDebounce'
import { useSearchSuggestions, usePopularSearches } from '@/features/search/hooks'
import { useRecentSearchesStore } from '@/stores/recentSearchesStore'
import { analytics } from '@/services/analytics'

export function SearchBar({ onNavigate }: { onNavigate?: () => void }) {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const debouncedQuery = useDebounce(query, 200)

  const { data: suggestions = [] } = useSearchSuggestions(debouncedQuery)
  const { data: popular = [] } = usePopularSearches()
  const recent = useRecentSearchesStore(s => s.queries)
  const recordRecent = useRecentSearchesStore(s => s.record)

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [])

  function runSearch(q: string) {
    const trimmed = q.trim()
    if (!trimmed) return
    recordRecent(trimmed)
    analytics.track('search', { query: trimmed })
    navigate(`/search?q=${encodeURIComponent(trimmed)}`)
    setOpen(false)
    onNavigate?.()
  }

  return (
    <div ref={containerRef} className="relative w-full">
      <form
        role="search"
        onSubmit={(e) => { e.preventDefault(); runSearch(query) }}
      >
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onFocus={() => setOpen(true)}
            placeholder="Search products, categories, brands…"
            aria-label="Search"
            className="pl-9 pr-9"
          />
          {query && (
            <button type="button" aria-label="Clear search" onClick={() => setQuery('')} className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </form>

      {open && (
        <div className="absolute z-50 mt-2 w-full rounded-[var(--radius)] border border-border bg-card p-2 shadow-lg animate-fade-in">
          {query.trim() && suggestions.length > 0 && (
            <div className="mb-2">
              <p className="px-2 py-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Suggestions</p>
              {suggestions.map(s => (
                <button
                  key={`${s.type}-${s.label}`}
                  type="button"
                  onClick={() => { recordRecent(query); navigate(s.href); setOpen(false); onNavigate?.() }}
                  className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm hover:bg-secondary"
                >
                  <span>{s.label}</span>
                  <span className="text-xs capitalize text-muted-foreground">{s.type}</span>
                </button>
              ))}
            </div>
          )}
          {!query.trim() && recent.length > 0 && (
            <div className="mb-2">
              <p className="px-2 py-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Recent searches</p>
              {recent.map(q => (
                <button key={q} type="button" onClick={() => runSearch(q)} className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-secondary">
                  <Clock className="h-3.5 w-3.5 text-muted-foreground" /> {q}
                </button>
              ))}
            </div>
          )}
          {!query.trim() && (
            <div>
              <p className="px-2 py-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Popular searches</p>
              {popular.map(q => (
                <button key={q} type="button" onClick={() => runSearch(q)} className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-secondary">
                  <TrendingUp className="h-3.5 w-3.5 text-muted-foreground" /> {q}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
