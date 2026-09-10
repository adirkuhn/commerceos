import type { Product } from './catalog'
import type { Category } from './catalog'
import type { Brand } from './catalog'
import type { CmsPage } from './content'

export interface SearchFilters {
  query: string
  categoryId?: string
  brandId?: string
  minPrice?: number
  maxPrice?: number
  minRating?: number
  inStockOnly?: boolean
  onSaleOnly?: boolean
  sort?: 'relevance' | 'price_asc' | 'price_desc' | 'newest' | 'rating' | 'best_selling'
  page?: number
  perPage?: number
}

export interface SearchResult {
  query: string
  products: Product[]
  categories: Category[]
  brands: Brand[]
  pages: CmsPage[]
  total: number
  page: number
  perPage: number
  suggestedQuery?: string // fuzzy "did you mean" correction
}

export interface SearchSuggestion {
  type: 'product' | 'category' | 'brand' | 'query'
  label: string
  href: string
}
