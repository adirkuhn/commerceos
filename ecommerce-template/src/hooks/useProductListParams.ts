import { useSearchParams } from 'react-router-dom'
import { useCallback, useMemo } from 'react'
import type { SearchFilters } from '@/types'
import type { FilterState } from '@/components/ecommerce/ProductFilters'

// Every filter/sort/page/view control reads from and writes to the URL's
// own query string -- shareable/bookmarkable filtered views, and browser
// back/forward moves between filter states, per spec section 2 ("Filters
// must work together and update the URL query parameters").
export function useProductListParams() {
  const [params, setParams] = useSearchParams()

  const filters: FilterState = useMemo(() => ({
    brandId: params.get('brand') ?? undefined,
    minPrice: params.get('min') ? Number(params.get('min')) : undefined,
    maxPrice: params.get('max') ? Number(params.get('max')) : undefined,
    minRating: params.get('rating') ? Number(params.get('rating')) : undefined,
    inStockOnly: params.get('inStock') === 'true',
    onSaleOnly: params.get('onSale') === 'true',
  }), [params])

  const sort = (params.get('sort') as SearchFilters['sort']) ?? 'relevance'
  const page = params.get('page') ? Number(params.get('page')) : 1
  const view = (params.get('view') as 'grid' | 'list') ?? 'grid'

  const setFilters = useCallback((next: FilterState) => {
    setParams(prev => {
      const p = new URLSearchParams(prev)
      next.brandId ? p.set('brand', next.brandId) : p.delete('brand')
      next.minPrice !== undefined ? p.set('min', String(next.minPrice)) : p.delete('min')
      next.maxPrice !== undefined ? p.set('max', String(next.maxPrice)) : p.delete('max')
      next.minRating !== undefined ? p.set('rating', String(next.minRating)) : p.delete('rating')
      next.inStockOnly ? p.set('inStock', 'true') : p.delete('inStock')
      next.onSaleOnly ? p.set('onSale', 'true') : p.delete('onSale')
      p.delete('page')
      return p
    })
  }, [setParams])

  const setSort = useCallback((next: SearchFilters['sort']) => {
    setParams(prev => {
      const p = new URLSearchParams(prev)
      next && next !== 'relevance' ? p.set('sort', next) : p.delete('sort')
      p.delete('page')
      return p
    })
  }, [setParams])

  const setPage = useCallback((next: number) => {
    setParams(prev => {
      const p = new URLSearchParams(prev)
      next > 1 ? p.set('page', String(next)) : p.delete('page')
      return p
    })
  }, [setParams])

  const setView = useCallback((next: 'grid' | 'list') => {
    setParams(prev => {
      const p = new URLSearchParams(prev)
      next === 'list' ? p.set('view', 'list') : p.delete('view')
      return p
    })
  }, [setParams])

  return { filters, setFilters, sort, setSort, page, setPage, view, setView }
}
