import { useParams } from 'react-router-dom'
import { Container } from '@/components/layout/Container'
import { Breadcrumbs } from '@/components/layout/Breadcrumbs'
import { Seo } from '@/components/seo/Seo'
import { NotFound, LoadingState } from '@/components/ui/States'
import { ProductListLayout, PER_PAGE } from '@/components/ecommerce/ProductListLayout'
import { useCategory } from '@/features/categories/hooks'
import { useProducts } from '@/features/products/hooks'
import { useProductListParams } from '@/hooks/useProductListParams'
import { analytics } from '@/services/analytics'
import { useEffect } from 'react'

export default function CategoryPage() {
  const { slug } = useParams<{ slug: string }>()
  const { data: category, isLoading: categoryLoading } = useCategory(slug)
  const { filters, sort, page } = useProductListParams()

  const { data, isLoading } = useProducts({
    categoryId: category?.id, brandId: filters.brandId, minPrice: filters.minPrice, maxPrice: filters.maxPrice,
    minRating: filters.minRating, inStockOnly: filters.inStockOnly, onSaleOnly: filters.onSaleOnly, sort, page, perPage: PER_PAGE,
  })

  useEffect(() => {
    if (category) analytics.track('category_viewed', { categoryId: category.id })
  }, [category])

  if (categoryLoading) return <LoadingState />
  if (!category) return <NotFound title="Category not found" description="This category doesn't exist or may have been removed." />

  return (
    <>
      <Seo title={category.name} description={category.description} />
      <Container className="py-8">
        <Breadcrumbs items={[{ label: category.name }]} />
        <h1 className="mb-2 mt-4 text-3xl font-semibold tracking-tight">{category.name}</h1>
        {category.description && <p className="mb-8 max-w-2xl text-muted-foreground">{category.description}</p>}
        <ProductListLayout products={data?.items ?? []} total={data?.total ?? 0} isLoading={isLoading} />
      </Container>
    </>
  )
}
