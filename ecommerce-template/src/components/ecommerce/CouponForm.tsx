import { useState } from 'react'
import { X } from 'lucide-react'
import type { AppliedCoupon } from '@/types'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { formatMoney } from '@/lib/utils'
import { useApplyCoupon, useRemoveCoupon } from '@/features/cart/hooks'

export function CouponForm({ coupons, currency }: { coupons: AppliedCoupon[]; currency: string }) {
  const [code, setCode] = useState('')
  const applyCoupon = useApplyCoupon()
  const removeCoupon = useRemoveCoupon()

  return (
    <div className="space-y-2">
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (!code.trim()) return
          applyCoupon.mutate(code, { onSuccess: () => setCode('') })
        }}
      >
        <Input placeholder="Coupon or gift card code" value={code} onChange={(e) => setCode(e.target.value)} aria-label="Coupon code" />
        <Button type="submit" variant="outline" disabled={applyCoupon.isPending || !code.trim()}>
          {applyCoupon.isPending ? 'Applying…' : 'Apply'}
        </Button>
      </form>
      {coupons.map(c => (
        <div key={c.code} className="flex items-center justify-between rounded-[var(--radius)] bg-accent px-3 py-2 text-sm text-accent-foreground">
          <span>{c.label} <span className="font-mono text-xs">({c.code})</span> -{formatMoney(c.discountAmount, currency)}</span>
          <button type="button" aria-label={`Remove coupon ${c.code}`} onClick={() => removeCoupon.mutate(c.code)}>
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  )
}
