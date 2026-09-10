import { useEffect, useRef, useState } from 'react'
import { LayoutGrid, List } from 'lucide-react'
import type { Product, SearchFilters } from '@/types'
import { ProductGrid } from './ProductGrid'
import { ProductFilterSidebar, ActiveFilterChips, SortSelect, type FilterState } from './ProductFilters'
import { Pagination } from '@/components/ui/Pagination'
import { Button } from '@/components/ui/Button'
import { ProductGridSkeleton } from '@/components/ui/States'
import { useBrands } from '@/features/categories/hooks'
import { useProductListParams } from '@/hooks/useProductListParams'

const PER_PAGE = 12

export function ProductListLayout({
  products, total, isLoading, emptyQuery, highlightQuery,
}: { products: Product[]; total: number; isLoading: boolean; emptyQuery?: string; highlightQuery?: string }) {
  const { filters, setFilters, sort, setSort, page, setPage, view, setView } = useProductListParams()
  const { data: brands = [] } = useBrands()
  const [infiniteScroll, setInfiniteScroll] = useState(false)
  const [accumulated, setAccumulated] = useState<Product[]>([])
  const sentinelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!infiniteScroll) return
    setAccumulated(prev => (page === 1 ? products : dedupe(prev, products)))
  }, [products, page, infiniteScroll])

  useEffect(() => {
    if (!infiniteScroll || !sentinelRef.current) return
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && page * PER_PAGE < total && !isLoading) setPage(page + 1)
    })
    observer.observe(sentinelRef.current)
    return () => observer.disconnect()
  }, [infiniteScroll, page, total, isLoading, setPage])

  const displayed = infiniteScroll ? accumulated : products

  return (
    <div className="grid gap-8 lg:grid-cols-[16rem_1fr]">
      <aside className="hidden lg:block">
        <ProductFilterSidebar filters={filters} onChange={setFilters} />
      </aside>
      <div>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <p className="text-sm text-muted-foreground">{total} {total === 1 ? 'product' : 'products'}</p>
            <ActiveFilterChips filters={filters} onChange={setFilters} brandLabel={brands.find(b => b.id === filters.brandId)?.name} />
          </div>
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <input type="checkbox" checked={infiniteScroll} onChange={(e) => { setInfiniteScroll(e.target.checked); setPage(1) }} />
              Infinite scroll
            </label>
            <SortSelect value={sort as SearchFilters['sort']} onChange={setSort} />
            <div className="flex items-center rounded-[var(--radius)] border border-input">
              <Button type="button" variant={view === 'grid' ? 'secondary' : 'ghost'} size="icon" aria-label="Grid view" aria-pressed={view === 'grid'} onClick={() => setView('grid')}>
                <LayoutGrid className="h-4 w-4" />
              </Button>
              <Button type="button" variant={view === 'list' ? 'secondary' : 'ghost'} size="icon" aria-label="List view" aria-pressed={view === 'list'} onClick={() => setView('list')}>
                <List className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        {isLoading && displayed.length === 0 ? <ProductGridSkeleton /> : <ProductGrid products={displayed} view={view} emptyQuery={emptyQuery} highlightQuery={highlightQuery} />}

        {infiniteScroll ? (
          <div ref={sentinelRef} className="h-8" />
        ) : (
          <div className="mt-8"><Pagination page={page} perPage={PER_PAGE} totalItems={total} onPageChange={setPage} /></div>
        )}
      </div>
    </div>
  )
}

function dedupe(a: Product[], b: Product[]): Product[] {
  const ids = new Set(a.map(p => p.id))
  return [...a, ...b.filter(p => !ids.has(p.id))]
}

export { PER_PAGE }
