import { useState } from 'react'
import { Link } from 'react-router-dom'
import type { Product } from '@/types'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog'
import { ProductPrice } from './ProductPrice'
import { ProductRating } from './ProductRating'
import { VariantSelector, findMatchingVariant } from './VariantSelector'
import { AddToCartButton } from './AddToCartButton'

export function QuickViewDialog({ product, open, onOpenChange }: { product: Product; open: boolean; onOpenChange: (open: boolean) => void }) {
  const [selected, setSelected] = useState<Record<string, string>>({})
  const variant = findMatchingVariant(product, selected)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <div className="grid gap-6 sm:grid-cols-2">
          <img src={product.images[0]?.url} alt={product.images[0]?.alt ?? product.name} className="aspect-square w-full rounded-[var(--radius)] object-cover" />
          <div className="space-y-3">
            <DialogTitle>{product.name}</DialogTitle>
            <DialogDescription>{product.shortDescription}</DialogDescription>
            <ProductRating rating={product.rating} reviewCount={product.reviewCount} />
            <ProductPrice price={variant?.price ?? product.price} compareAtPrice={variant?.compareAtPrice ?? product.compareAtPrice} currency={product.currency} size="lg" />
            <VariantSelector product={product} selected={selected} onChange={(name, value) => setSelected(s => ({ ...s, [name]: value }))} />
            <div className="flex items-center gap-3 pt-2">
              <AddToCartButton product={product} variant={variant} disabled={product.variantOptionNames.length > 0 && !variant} />
              <Link to={`/product/${product.slug}`} className="text-sm text-primary hover:underline" onClick={() => onOpenChange(false)}>
                View full details
              </Link>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
