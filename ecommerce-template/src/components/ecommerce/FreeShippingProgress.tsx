import { formatMoney } from '@/lib/utils'
import type { CartTotals } from '@/types'

export function FreeShippingProgress({ totals, currency }: { totals: CartTotals; currency: string }) {
  const progress = Math.min(100, ((totals.freeShippingThreshold - totals.freeShippingRemaining) / totals.freeShippingThreshold) * 100)

  return (
    <div className="space-y-1.5">
      <p className="text-xs text-muted-foreground">
        {totals.freeShippingRemaining > 0
          ? <>Add <strong className="text-foreground">{formatMoney(totals.freeShippingRemaining, currency)}</strong> more for free shipping</>
          : <span className="font-medium text-emerald-600">You've unlocked free shipping!</span>}
      </p>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary" role="progressbar" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100}>
        <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${progress}%` }} />
      </div>
    </div>
  )
}
