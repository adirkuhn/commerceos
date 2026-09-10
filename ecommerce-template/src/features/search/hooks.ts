import { useQuery } from '@tanstack/react-query'
import { search, getSearchSuggestions, getPopularSearches } from '@/api/search'
import type { SearchFilters } from '@/types'

export function useSearch(filters: SearchFilters) {
  return useQuery({ queryKey: ['search', filters], queryFn: () => search(filters), enabled: filters.query.trim().length > 0 || Object.keys(filters).length > 2 })
}

export function useSearchSuggestions(query: string) {
  return useQuery({ queryKey: ['search-suggestions', query], queryFn: () => getSearchSuggestions(query), enabled: query.trim().length > 0 })
}

export function usePopularSearches() {
  return useQuery({ queryKey: ['popular-searches'], queryFn: getPopularSearches, staleTime: 10 * 60 * 1000 })
}
