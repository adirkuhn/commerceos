import { searchRepository } from '@/repositories'
import type { SearchFilters } from '@/types'

export const search = (filters: SearchFilters) => searchRepository.search(filters)
export const getSearchSuggestions = (query: string) => searchRepository.suggestions(query)
export const getPopularSearches = () => searchRepository.popularSearches()
