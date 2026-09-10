import { Link } from 'react-router-dom'
import type { ProductGridSection as ProductGridSectionType } from '@/types'
import { useProducts, useProductsByIds } from '@/features/products/hooks'
import { useCategories } from '@/features/categories/hooks'
import { ProductGrid } from '@/components/ecommerce/ProductGrid'
import { ProductGridSkeleton } from '@/components/ui/States'

export function ProductGridSection({ section }: { section: ProductGridSectionType }) {
  const byIds = useProductsByIds(section.productIds ?? [])
  const byFilter = useProducts({ categoryId: section.categoryId, filter: section.filter, perPage: section.limit ?? 8 })
  const { data, isLoading } = section.productIds && section.productIds.length > 0 ? byIds : { data: byFilter.data?.items, isLoading: byFilter.isLoading }
  const { data: categories = [] } = useCategories()
  const categorySlug = categories.find(c => c.id === section.categoryId)?.slug

  return (
    <section>
      <div className="mb-6 flex items-end justify-between">
        <div>
          <h2 className="text-xl font-semibold tracking-tight sm:text-2xl">{section.title}</h2>
          {section.subtitle && <p className="mt-1 text-muted-foreground">{section.subtitle}</p>}
        </div>
        {categorySlug && <Link to={`/category/${categorySlug}`} className="text-sm font-medium text-primary hover:underline">View all</Link>}
      </div>
      {isLoading ? <ProductGridSkeleton count={section.limit ?? 8} /> : <ProductGrid products={data ?? []} />}
    </section>
  )
}
