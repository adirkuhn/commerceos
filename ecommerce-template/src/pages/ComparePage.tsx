import { Link } from 'react-router-dom'
import { Scale, X } from 'lucide-react'
import { Container } from '@/components/layout/Container'
import { PageHeader } from '@/components/layout/PageHeader'
import { Seo } from '@/components/seo/Seo'
import { Button } from '@/components/ui/Button'
import { EmptyState, LoadingState } from '@/components/ui/States'
import { ProductPrice } from '@/components/ecommerce/ProductPrice'
import { ProductRating } from '@/components/ecommerce/ProductRating'
import { AddToCartButton } from '@/components/ecommerce/AddToCartButton'
import { useCompareStore } from '@/stores/compareStore'
import { useProductsByIds } from '@/features/products/hooks'
import { productStockLabel } from '@/types'

const ATTRIBUTE_ROWS = ['Brand', 'Price', 'Rating', 'Availability', 'Attributes'] as const

export default function ComparePage() {
  const productIds = useCompareStore(s => s.productIds)
  const remove = useCompareStore(s => s.remove)
  const { data: products, isLoading } = useProductsByIds(productIds)

  return (
    <>
      <Seo title="Compare products" />
      <Container className="pb-16">
        <PageHeader title="Compare products" description="Compare up to 4 products side by side." />
        {isLoading && <LoadingState />}
        {!isLoading && (!products || products.length === 0) && (
          <EmptyState
            icon={<Scale className="h-8 w-8" />}
            title="No products to compare"
            description="Add products from any product page using the compare icon."
            action={<Button asChild className="mt-2"><Link to="/">Browse products</Link></Button>}
          />
        )}
        {products && products.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[640px] border-collapse text-left text-sm">
              <thead>
                <tr>
                  <th className="w-32" />
                  {products.map(p => (
                    <th key={p.id} className="p-3 align-top">
                      <button aria-label={`Remove ${p.name} from comparison`} onClick={() => remove(p.id)} className="mb-2 text-muted-foreground hover:text-foreground">
                        <X className="h-4 w-4" />
                      </button>
                      <img src={p.images[0]?.url} alt={p.name} className="mb-2 aspect-square w-full rounded-[var(--radius)] object-cover" />
                      <Link to={`/product/${p.slug}`} className="font-medium hover:underline">{p.name}</Link>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {ATTRIBUTE_ROWS.map(row => (
                  <tr key={row}>
                    <th scope="row" className="p-3 align-top font-medium text-muted-foreground">{row}</th>
                    {products.map(p => (
                      <td key={p.id} className="p-3 align-top">
                        {row === 'Brand' && (p.brandId ?? '—')}
                        {row === 'Price' && <ProductPrice price={p.price} compareAtPrice={p.compareAtPrice} currency={p.currency} size="sm" />}
                        {row === 'Rating' && <ProductRating rating={p.rating} reviewCount={p.reviewCount} />}
                        {row === 'Availability' && productStockLabel(p)}
                        {row === 'Attributes' && (
                          <ul className="space-y-0.5 text-xs text-muted-foreground">
                            {p.attributes.length === 0 ? <li>—</li> : p.attributes.map(a => <li key={a.name}>{a.name}: {a.value}</li>)}
                          </ul>
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
                <tr>
                  <th scope="row" className="p-3" />
                  {products.map(p => <td key={p.id} className="p-3"><AddToCartButton product={p} size="sm" /></td>)}
                </tr>
              </tbody>
            </table>
          </div>
        )}
      </Container>
    </>
  )
}
