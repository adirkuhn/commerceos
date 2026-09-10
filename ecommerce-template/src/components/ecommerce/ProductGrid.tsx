import type { Product } from '@/types'
import { ProductCard } from './ProductCard'
import { NoResultsState } from '@/components/ui/States'
import { cn } from '@/lib/utils'

export function ProductGrid({
  products, view = 'grid', emptyQuery, highlightQuery,
}: { products: Product[]; view?: 'grid' | 'list'; emptyQuery?: string; highlightQuery?: string }) {
  if (products.length === 0) return <NoResultsState query={emptyQuery} />

  return (
    <div className={cn(view === 'grid' ? 'grid grid-cols-2 gap-x-4 gap-y-8 sm:grid-cols-3 lg:grid-cols-4' : 'flex flex-col gap-6')}>
      {products.map(p => <ProductCard key={p.id} product={p} view={view} highlightQuery={highlightQuery} />)}
    </div>
  )
}
