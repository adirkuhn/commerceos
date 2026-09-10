import { useSearchParams, Link } from 'react-router-dom'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Badge } from '@/components/ui/Badge'
import { ProductListLayout, PER_PAGE } from '@/components/ecommerce/ProductListLayout'
import { useSearch } from '@/features/search/hooks'
import { useProductListParams } from '@/hooks/useProductListParams'

export default function SearchPage() {
  const [params] = useSearchParams()
  const query = params.get('q') ?? ''
  const { filters, sort, page } = useProductListParams()

  const { data, isLoading } = useSearch({
    query, brandId: filters.brandId, minPrice: filters.minPrice, maxPrice: filters.maxPrice,
    minRating: filters.minRating, inStockOnly: filters.inStockOnly, onSaleOnly: filters.onSaleOnly, sort, page, perPage: PER_PAGE,
  })

  return (
    <>
      <Seo title={query ? `Search: ${query}` : 'Search'} />
      <Container className="py-8">
        <h1 className="mb-1 text-2xl font-semibold tracking-tight">{query ? `Results for "${query}"` : 'Search'}</h1>
        {data?.suggestedQuery && (
          <p className="mb-4 text-sm text-muted-foreground">
            No exact matches. Did you mean{' '}
            <Link to={`/search?q=${encodeURIComponent(data.suggestedQuery)}`} className="text-primary underline">{data.suggestedQuery}</Link>?
          </p>
        )}

        {!!data && (data.categories.length > 0 || data.brands.length > 0) && (
          <div className="mb-6 flex flex-wrap gap-2">
            {data.categories.map(c => (
              <Link key={c.id} to={`/category/${c.slug}`}><Badge variant="outline">Category: {c.name}</Badge></Link>
            ))}
            {data.brands.map(b => <Badge key={b.id} variant="outline">Brand: {b.name}</Badge>)}
          </div>
        )}

        <ProductListLayout products={data?.products ?? []} total={data?.total ?? 0} isLoading={isLoading} emptyQuery={query} highlightQuery={query} />
      </Container>
    </>
  )
}
