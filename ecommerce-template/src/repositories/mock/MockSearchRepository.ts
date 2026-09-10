import type { SearchFilters, SearchResult, SearchSuggestion } from '@/types'
import type { SearchRepository } from '../types'
import { simulateLatency } from './latency'
import { PRODUCTS, CATEGORIES, BRANDS, CMS_PAGES } from '@/config/demoData'
import { levenshtein } from '@/lib/utils'

const POPULAR_SEARCHES = ['trail runner', 'rain jacket', 'daypack', 'merino tee', 'headlamp']

function textMatches(haystack: string, query: string): boolean {
  return haystack.toLowerCase().includes(query.toLowerCase())
}

function closestWord(query: string): string | undefined {
  const words = new Set<string>()
  for (const p of PRODUCTS) {
    for (const w of p.name.split(' ')) words.add(w)
  }
  let best: string | undefined
  let bestDistance = Infinity
  for (const word of words) {
    const distance = levenshtein(query.toLowerCase(), word.toLowerCase())
    if (distance < bestDistance) {
      bestDistance = distance
      best = word
    }
  }
  // Only suggest a correction for a plausible typo, not an unrelated word.
  return bestDistance > 0 && bestDistance <= 3 ? best : undefined
}

export class MockSearchRepository implements SearchRepository {
  async search(filters: SearchFilters): Promise<SearchResult> {
    await simulateLatency()
    const query = filters.query.trim()

    let products = PRODUCTS.filter(p => {
      if (query && !(textMatches(p.name, query) || textMatches(p.shortDescription, query) || p.tags.some(t => textMatches(t, query)))) return false
      if (filters.categoryId && !p.categoryIds.includes(filters.categoryId)) return false
      if (filters.brandId && p.brandId !== filters.brandId) return false
      if (filters.minPrice !== undefined && p.price < filters.minPrice) return false
      if (filters.maxPrice !== undefined && p.price > filters.maxPrice) return false
      if (filters.minRating !== undefined && p.rating < filters.minRating) return false
      if (filters.inStockOnly && p.stockStatus === 'out_of_stock') return false
      if (filters.onSaleOnly && !p.compareAtPrice) return false
      return true
    })

    switch (filters.sort) {
      case 'price_asc': products = [...products].sort((a, b) => a.price - b.price); break
      case 'price_desc': products = [...products].sort((a, b) => b.price - a.price); break
      case 'rating': products = [...products].sort((a, b) => b.rating - a.rating); break
      case 'newest': products = [...products].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()); break
      default: break
    }

    const categories = query ? CATEGORIES.filter(c => textMatches(c.name, query)) : []
    const brands = query ? BRANDS.filter(b => textMatches(b.name, query)) : []
    const pages = query ? CMS_PAGES.filter(p => textMatches(p.title, query)) : []

    const total = products.length
    const page = filters.page ?? 1
    const perPage = filters.perPage ?? 12
    const start = (page - 1) * perPage

    return {
      query,
      products: products.slice(start, start + perPage),
      categories,
      brands,
      pages,
      total,
      page,
      perPage,
      suggestedQuery: query && total === 0 ? closestWord(query) : undefined,
    }
  }

  async suggestions(query: string): Promise<SearchSuggestion[]> {
    await simulateLatency()
    if (!query.trim()) return []
    const suggestions: SearchSuggestion[] = []
    for (const p of PRODUCTS) {
      if (textMatches(p.name, query) && suggestions.length < 5) {
        suggestions.push({ type: 'product', label: p.name, href: `/product/${p.slug}` })
      }
    }
    for (const c of CATEGORIES) {
      if (textMatches(c.name, query) && suggestions.filter(s => s.type === 'category').length < 2) {
        suggestions.push({ type: 'category', label: c.name, href: `/category/${c.slug}` })
      }
    }
    for (const b of BRANDS) {
      if (textMatches(b.name, query) && suggestions.filter(s => s.type === 'brand').length < 2) {
        suggestions.push({ type: 'brand', label: b.name, href: `/search?brand=${b.id}` })
      }
    }
    return suggestions
  }

  async popularSearches(): Promise<string[]> {
    await simulateLatency()
    return POPULAR_SEARCHES
  }
}
