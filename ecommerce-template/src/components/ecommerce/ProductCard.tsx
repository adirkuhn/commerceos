import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Eye } from 'lucide-react'
import type { Product } from '@/types'
import { ProductPrice } from './ProductPrice'
import { ProductRating } from './ProductRating'
import { ProductBadges } from './ProductBadge'
import { WishlistButton } from './WishlistButton'
import { CompareButton } from './CompareButton'
import { AddToCartButton } from './AddToCartButton'
import { QuickViewDialog } from './QuickViewDialog'
import { Button } from '@/components/ui/Button'
import { analytics } from '@/services/analytics'
import { highlightMatch } from '@/lib/utils'

export function ProductCard({ product, view = 'grid', highlightQuery }: { product: Product; view?: 'grid' | 'list'; highlightQuery?: string }) {
  const [quickViewOpen, setQuickViewOpen] = useState(false)
  const [hovering, setHovering] = useState(false)
  const primaryImage = product.images[0]
  const hoverImage = product.images[1] ?? primaryImage
  const defaultVariant = product.variants[0]

  return (
    <div
      className={view === 'list' ? 'flex gap-4' : 'flex flex-col'}
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      <div className={view === 'list' ? 'relative w-40 shrink-0' : 'relative'}>
        <Link to={`/product/${product.slug}`} onClick={() => analytics.track('product_viewed', { productId: product.id, source: 'card' })}>
          <div className="aspect-square overflow-hidden rounded-[var(--radius)] bg-muted">
            <img
              src={(hovering ? hoverImage : primaryImage)?.url}
              alt={primaryImage?.alt ?? product.name}
              loading="lazy"
              className="h-full w-full object-cover transition-opacity"
            />
          </div>
        </Link>
        <div className="absolute left-2 top-2"><ProductBadges product={product} /></div>
        <div className="absolute right-2 top-2 flex flex-col gap-1.5">
          <WishlistButton productId={product.id} className="h-8 w-8 bg-card/90 backdrop-blur" />
          <CompareButton productId={product.id} className="h-8 w-8 bg-card/90 backdrop-blur" />
          <Button
            type="button" variant="outline" size="icon" className="h-8 w-8 bg-card/90 backdrop-blur"
            aria-label="Quick view" onClick={() => setQuickViewOpen(true)}
          >
            <Eye className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className={view === 'list' ? 'flex flex-1 flex-col justify-between' : 'mt-3 flex flex-1 flex-col gap-1.5'}>
        <div>
          <Link to={`/product/${product.slug}`} className="text-sm font-medium hover:underline">
            <HighlightedName name={product.name} query={highlightQuery} />
          </Link>
          <p className="text-xs text-muted-foreground">SKU {product.sku}</p>
          <ProductRating rating={product.rating} reviewCount={product.reviewCount} />
          <ProductPrice price={product.price} compareAtPrice={product.compareAtPrice} currency={product.currency} className="mt-1" />
        </div>
        <div className="mt-2 flex gap-2">
          <AddToCartButton product={product} variant={product.variantOptionNames.length === 0 ? undefined : defaultVariant} size="sm" label="Quick add" />
        </div>
      </div>

      <QuickViewDialog product={product} open={quickViewOpen} onOpenChange={setQuickViewOpen} />
    </div>
  )
}

function HighlightedName({ name, query }: { name: string; query?: string }) {
  const match = query ? highlightMatch(name, query) : null
  if (!match) return <>{name}</>
  return <>{match.before}<mark className="rounded-sm bg-accent text-accent-foreground">{match.match}</mark>{match.after}</>
}
