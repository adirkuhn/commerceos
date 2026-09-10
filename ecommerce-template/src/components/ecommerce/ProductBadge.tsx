import type { Product } from '@/types'
import { Badge } from '@/components/ui/Badge'

export function ProductBadges({ product }: { product: Product }) {
  const badges: { label: string; variant: 'default' | 'secondary' | 'sale' | 'outline' }[] = []
  if (product.compareAtPrice && product.compareAtPrice > product.price) badges.push({ label: 'Sale', variant: 'sale' })
  if (product.isNewArrival) badges.push({ label: 'New', variant: 'secondary' })
  if (product.isBestSeller) badges.push({ label: 'Best seller', variant: 'default' })
  if (product.stockStatus === 'low_stock') badges.push({ label: 'Low stock', variant: 'outline' })
  if (product.stockStatus === 'out_of_stock') badges.push({ label: 'Out of stock', variant: 'outline' })

  if (badges.length === 0) return null
  return (
    <div className="flex flex-wrap gap-1">
      {badges.map(b => <Badge key={b.label} variant={b.variant}>{b.label}</Badge>)}
    </div>
  )
}
