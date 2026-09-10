import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Share2, FileText, AlertTriangle } from 'lucide-react'
import { Container } from '@/components/layout/Container'
import { Breadcrumbs } from '@/components/layout/Breadcrumbs'
import { Seo } from '@/components/seo/Seo'
import { LoadingState, NotFound } from '@/components/ui/States'
import { Button } from '@/components/ui/Button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/Tabs'
import { Badge } from '@/components/ui/Badge'
import { ProductGallery } from '@/components/ecommerce/ProductGallery'
import { B2BPriceDisplay } from '@/components/ecommerce/B2BPriceDisplay'
import { QuantityBreaksTable } from '@/components/ecommerce/QuantityBreaksTable'
import { ProductRating } from '@/components/ecommerce/ProductRating'
import { VariantSelector, findMatchingVariant } from '@/components/ecommerce/VariantSelector'
import { QuantitySelector } from '@/components/ecommerce/QuantitySelector'
import { AddToCartButton } from '@/components/ecommerce/AddToCartButton'
import { WishlistButton } from '@/components/ecommerce/WishlistButton'
import { CompareButton } from '@/components/ecommerce/CompareButton'
import { ShippingEstimator } from '@/components/ecommerce/ShippingEstimator'
import { ReviewsSection } from '@/components/ecommerce/ReviewsSection'
import { QASection } from '@/components/ecommerce/QASection'
import { ProductGrid } from '@/components/ecommerce/ProductGrid'
import { RecentlyViewedRail } from '@/components/ecommerce/RecentlyViewedRail'
import { useProduct, useProductsByIds } from '@/features/products/hooks'
import { useCategories, useBrands } from '@/features/categories/hooks'
import { useCart, useAddCartItem } from '@/features/cart/hooks'
import { useRecentlyViewedStore } from '@/stores/recentlyViewedStore'
import { productStockLabel } from '@/types'
import { toast } from '@/components/ui/toast'
import { analytics } from '@/services/analytics'
import { pluginRegistry } from '@/plugins'

export default function ProductDetailPage() {
  const { slug } = useParams<{ slug: string }>()
  const { data: product, isLoading } = useProduct(slug)
  const { data: categories = [] } = useCategories()
  const { data: brands = [] } = useBrands()
  const recordViewed = useRecentlyViewedStore(s => s.record)

  const [selected, setSelected] = useState<Record<string, string>>({})
  const [quantity, setQuantity] = useState(1)
  const variant = product ? findMatchingVariant(product, selected) : undefined
  const addItem = useAddCartItem()
  const { data: cart } = useCart()

  useEffect(() => {
    if (!product) return
    recordViewed(product.id)
    analytics.track('product_viewed', { productId: product.id })
    if (product.minOrderQuantity) setQuantity(product.minOrderQuantity)
  }, [product, recordViewed])

  if (isLoading) return <LoadingState />
  if (!product) return <NotFound title="Product not found" description="This product may have sold out or been removed." />

  const category = categories.find(c => c.id === product.categoryIds[0])
  const brand = brands.find(b => b.id === product.brandId)
  const price = variant?.price ?? product.price
  const compareAtPrice = variant?.compareAtPrice ?? product.compareAtPrice
  const stock = variant?.stock ?? product.stock
  const needsVariant = product.variantOptionNames.length > 0 && !variant
  const alreadyInCart = cart?.items.some(i => i.productId === product.id && i.variantId === variant?.id) ?? false
  const pluginTabs = pluginRegistry.productTabs()

  return (
    <>
      <Seo
        title={product.name}
        description={product.shortDescription}
        image={product.images[0]?.url}
        jsonLd={[
          {
            '@context': 'https://schema.org', '@type': 'Product', name: product.name, sku: product.sku,
            image: product.images.map(i => i.url), description: product.shortDescription,
            brand: brand ? { '@type': 'Brand', name: brand.name } : undefined,
            aggregateRating: product.reviewCount > 0 ? { '@type': 'AggregateRating', ratingValue: product.rating, reviewCount: product.reviewCount } : undefined,
            offers: { '@type': 'Offer', price: (price / 100).toFixed(2), priceCurrency: product.currency, availability: stock > 0 ? 'https://schema.org/InStock' : 'https://schema.org/OutOfStock' },
          },
          {
            '@context': 'https://schema.org', '@type': 'BreadcrumbList',
            itemListElement: [
              category && { '@type': 'ListItem', position: 1, name: category.name, item: `/category/${category.slug}` },
              { '@type': 'ListItem', position: 2, name: product.name },
            ].filter(Boolean),
          },
        ]}
      />
      <Container className="space-y-16 py-8">
        <div>
          <Breadcrumbs items={[category && { label: category.name, href: `/category/${category.slug}` }, { label: product.name }].filter(Boolean) as { label: string; href?: string }[]} />

          <div className="mt-6 grid gap-10 lg:grid-cols-2">
            <ProductGallery images={product.images} alt={product.name} />

            <div>
              {brand && <Link to={`/search?brand=${brand.id}`} className="text-sm text-muted-foreground hover:underline">{brand.name}</Link>}
              <h1 className="mt-1 text-2xl font-semibold tracking-tight sm:text-3xl">{product.name}</h1>
              <p className="mt-1 text-xs text-muted-foreground">SKU {variant?.sku ?? product.sku}</p>
              <div className="mt-2"><ProductRating rating={product.rating} reviewCount={product.reviewCount} /></div>
              <B2BPriceDisplay price={price} compareAtPrice={compareAtPrice} currency={product.currency} size="lg" className="mt-4" />
              <p className="mt-1 text-xs text-muted-foreground">{product.taxIncluded ? 'Tax included' : 'Tax calculated at checkout'}</p>

              <p className="mt-4 text-sm text-muted-foreground">{product.shortDescription}</p>

              {product.variantOptionNames.length > 0 && (
                <div className="mt-6"><VariantSelector product={product} selected={selected} onChange={(name, value) => setSelected(s => ({ ...s, [name]: value }))} /></div>
              )}

              <div className="mt-4">
                <p className={stock <= product.lowStockThreshold && stock > 0 ? 'text-sm font-medium text-amber-600' : 'text-sm text-muted-foreground'}>
                  {stock <= product.lowStockThreshold && stock > 0 && <AlertTriangle className="mr-1 inline h-3.5 w-3.5" />}
                  {productStockLabel({ ...product, stock, stockStatus: stock === 0 ? 'out_of_stock' : stock <= product.lowStockThreshold ? 'low_stock' : 'in_stock' })}
                </p>
              </div>

              <div className="mt-4 flex items-center gap-4">
                <QuantitySelector value={quantity} onChange={setQuantity} min={product.minOrderQuantity ?? 1} max={Math.max(product.minOrderQuantity ?? 1, stock)} />
                <AddToCartButton product={product} variant={variant} quantity={quantity} disabled={needsVariant} />
              </div>
              <div className="mt-3 flex gap-3">
                <Button
                  variant="secondary"
                  disabled={needsVariant || stock === 0}
                  onClick={() => addItem.mutate(
                    { productId: product.id, variantId: variant?.id, name: product.name, variantLabel: variant?.options.map(o => o.value).join(' / '), imageUrl: variant?.imageUrl ?? product.images[0]?.url, sku: variant?.sku ?? product.sku, price, quantity },
                    { onSuccess: () => toast({ title: 'Redirecting to checkout would happen here', description: 'Buy Now added the item to your cart.' }) },
                  )}
                >
                  Buy now
                </Button>
                <WishlistButton productId={product.id} />
                <CompareButton productId={product.id} />
                <Button
                  type="button" variant="outline" size="icon" aria-label="Share this product"
                  onClick={async () => {
                    const url = window.location.href
                    if (navigator.share) await navigator.share({ title: product.name, url }).catch(() => {})
                    else { await navigator.clipboard.writeText(url); toast({ title: 'Link copied to clipboard' }) }
                  }}
                >
                  <Share2 className="h-4 w-4" />
                </Button>
              </div>
              {alreadyInCart && <p className="mt-2 text-xs text-emerald-600">Already in your cart</p>}

              <div className="mt-6"><ShippingEstimator /></div>

              {product.quantityBreaks && product.quantityBreaks.length > 0 && (
                <div className="mt-6">
                  <QuantityBreaksTable basePrice={price} currency={product.currency} minOrderQuantity={product.minOrderQuantity} tiers={product.quantityBreaks} />
                </div>
              )}

              {pluginTabs.length > 0 && (
                <div className="mt-6 space-y-4">
                  {pluginTabs.map(tab => {
                    const Component = tab.render(product)
                    return Component ? <div key={tab.id}><p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{tab.label}</p><Component /></div> : null
                  })}
                </div>
              )}
            </div>
          </div>
        </div>

        <Tabs defaultValue="description">
          <TabsList>
            <TabsTrigger value="description">Description</TabsTrigger>
            <TabsTrigger value="specs">Specifications</TabsTrigger>
            {product.documents && product.documents.length > 0 && <TabsTrigger value="documents">Documents</TabsTrigger>}
            <TabsTrigger value="reviews">Reviews ({product.reviewCount})</TabsTrigger>
            <TabsTrigger value="qa">Q&amp;A</TabsTrigger>
          </TabsList>
          <TabsContent value="description">
            <p className="max-w-2xl text-sm leading-relaxed text-muted-foreground">{product.description}</p>
            {product.tags.length > 0 && (
              <div className="mt-4 flex flex-wrap gap-1.5">{product.tags.map(t => <Badge key={t} variant="outline">{t}</Badge>)}</div>
            )}
          </TabsContent>
          <TabsContent value="specs">
            {product.attributes.length === 0 ? (
              <p className="text-sm text-muted-foreground">No additional specifications for this product.</p>
            ) : (
              <dl className="grid max-w-md grid-cols-2 gap-2 text-sm">
                {product.attributes.map(a => (
                  <div key={a.name} className="contents">
                    <dt className="text-muted-foreground">{a.name}</dt><dd>{a.value}</dd>
                  </div>
                ))}
                {product.weightGrams && <><dt className="text-muted-foreground">Weight</dt><dd>{product.weightGrams}g</dd></>}
              </dl>
            )}
          </TabsContent>
          {product.documents && product.documents.length > 0 && (
            <TabsContent value="documents">
              <ul className="space-y-2">
                {product.documents.map(doc => (
                  <li key={doc.id}>
                    <a href={doc.url} className="flex items-center gap-2 text-sm text-primary hover:underline"><FileText className="h-4 w-4" /> {doc.label} ({doc.sizeLabel})</a>
                  </li>
                ))}
              </ul>
            </TabsContent>
          )}
          <TabsContent value="reviews"><ReviewsSection productId={product.id} /></TabsContent>
          <TabsContent value="qa"><QASection productId={product.id} /></TabsContent>
        </Tabs>

        <RelatedProducts title="Related products" productIds={product.relatedProductIds} />
        <RelatedProducts title="You may also like" productIds={product.crossSellProductIds} />
        <RelatedProducts title="Frequently bought together" productIds={product.frequentlyBoughtWithIds} />
        <RelatedProducts title="Complete the set" productIds={product.upsellProductIds} />

        {category && (
          <div className="flex flex-wrap gap-2">
            <span className="text-sm text-muted-foreground">Related categories:</span>
            {categories.filter(c => c.id !== category.id).slice(0, 3).map(c => (
              <Link key={c.id} to={`/category/${c.slug}`}><Badge variant="secondary">{c.name}</Badge></Link>
            ))}
          </div>
        )}

        <RecentlyViewedRail excludeProductId={product.id} />
      </Container>
    </>
  )
}

function RelatedProducts({ title, productIds }: { title: string; productIds?: string[] }) {
  const { data: products = [] } = useProductsByIds(productIds ?? [])
  if (products.length === 0) return null
  return (
    <section>
      <h2 className="mb-6 text-xl font-semibold tracking-tight">{title}</h2>
      <ProductGrid products={products} />
    </section>
  )
}
