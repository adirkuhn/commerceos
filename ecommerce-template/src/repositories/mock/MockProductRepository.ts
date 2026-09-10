import { PRODUCTS } from '@/config/demoData'
import type { Product } from '@/types'
import type { ProductListParams, ProductListResult, ProductRepository } from '../types'
import { simulateLatency } from './latency'

function matchesFilter(product: Product, filter: ProductListParams['filter']): boolean {
  switch (filter) {
    case 'featured': return product.isFeatured
    case 'new_arrivals': return product.isNewArrival
    case 'best_sellers': return product.isBestSeller
    case 'recommended': return product.rating >= 4
    default: return true
  }
}

function sortProducts(items: Product[], sort: ProductListParams['sort']): Product[] {
  const sorted = [...items]
  switch (sort) {
    case 'price_asc': return sorted.sort((a, b) => a.price - b.price)
    case 'price_desc': return sorted.sort((a, b) => b.price - a.price)
    case 'newest': return sorted.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    case 'rating': return sorted.sort((a, b) => b.rating - a.rating)
    case 'best_selling': return sorted.sort((a, b) => Number(b.isBestSeller) - Number(a.isBestSeller))
    default: return sorted
  }
}

export class MockProductRepository implements ProductRepository {
  async list(params: ProductListParams): Promise<ProductListResult> {
    await simulateLatency()
    let items = PRODUCTS.filter(p => {
      if (params.categoryId && !p.categoryIds.includes(params.categoryId)) return false
      if (params.brandId && p.brandId !== params.brandId) return false
      if (params.tag && !p.tags.includes(params.tag)) return false
      if (params.filter && !matchesFilter(p, params.filter)) return false
      if (params.excludeProductId && p.id === params.excludeProductId) return false
      if (params.ids && !params.ids.includes(p.id)) return false
      if (params.minPrice !== undefined && p.price < params.minPrice) return false
      if (params.maxPrice !== undefined && p.price > params.maxPrice) return false
      if (params.minRating !== undefined && p.rating < params.minRating) return false
      if (params.inStockOnly && p.stockStatus === 'out_of_stock') return false
      if (params.onSaleOnly && !p.compareAtPrice) return false
      return true
    })
    items = sortProducts(items, params.sort)

    const total = items.length
    const page = params.page ?? 1
    const perPage = params.perPage ?? total
    const start = (page - 1) * perPage
    return { items: items.slice(start, start + perPage), total }
  }

  async getBySlug(slug: string): Promise<Product | null> {
    await simulateLatency()
    return PRODUCTS.find(p => p.slug === slug) ?? null
  }

  async getById(id: string): Promise<Product | null> {
    await simulateLatency()
    return PRODUCTS.find(p => p.id === id) ?? null
  }

  async getByIds(ids: string[]): Promise<Product[]> {
    await simulateLatency()
    return PRODUCTS.filter(p => ids.includes(p.id))
  }
}
