import { useRecentlyViewedStore } from '@/stores/recentlyViewedStore'
import { useProductsByIds } from '@/features/products/hooks'
import { ProductGrid } from './ProductGrid'

export function RecentlyViewedRail({ excludeProductId }: { excludeProductId?: string }) {
  const productIds = useRecentlyViewedStore(s => s.productIds).filter(id => id !== excludeProductId)
  const { data: products = [] } = useProductsByIds(productIds)

  if (products.length === 0) return null

  return (
    <section>
      <h2 className="mb-6 text-xl font-semibold tracking-tight sm:text-2xl">Recently viewed</h2>
      <ProductGrid products={products} />
    </section>
  )
}
