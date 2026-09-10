import { formatMoney } from '@/lib/utils'
import { cn } from '@/lib/utils'

export function ProductPrice({
  price, compareAtPrice, currency = 'USD', size = 'default', className,
}: { price: number; compareAtPrice?: number; currency?: string; size?: 'sm' | 'default' | 'lg'; className?: string }) {
  const onSale = !!compareAtPrice && compareAtPrice > price
  const discountPercent = onSale ? Math.round((1 - price / compareAtPrice!) * 100) : 0
  const sizeClass = size === 'lg' ? 'text-2xl' : size === 'sm' ? 'text-sm' : 'text-base'

  return (
    <div className={cn('flex items-baseline gap-2', className)}>
      <span className={cn('font-semibold', sizeClass)}>{formatMoney(price, currency)}</span>
      {onSale && (
        <>
          <span className="text-sm text-muted-foreground line-through">{formatMoney(compareAtPrice!, currency)}</span>
          <span className="text-xs font-medium text-destructive">-{discountPercent}%</span>
        </>
      )}
    </div>
  )
}
