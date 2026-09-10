import type { CartTotals } from '@/types'
import { formatMoney } from '@/lib/utils'
import { Separator } from '@/components/ui/Separator'

export function CheckoutSummary({ totals, currency }: { totals: CartTotals; currency: string }) {
  return (
    <dl className="space-y-2 text-sm">
      <div className="flex justify-between"><dt className="text-muted-foreground">Subtotal</dt><dd>{formatMoney(totals.subtotal, currency)}</dd></div>
      {totals.discount > 0 && (
        <div className="flex justify-between text-emerald-600"><dt>Discount</dt><dd>-{formatMoney(totals.discount, currency)}</dd></div>
      )}
      <div className="flex justify-between"><dt className="text-muted-foreground">Shipping</dt><dd>{totals.shipping === 0 ? 'Free' : formatMoney(totals.shipping, currency)}</dd></div>
      <div className="flex justify-between"><dt className="text-muted-foreground">Tax</dt><dd>{formatMoney(totals.tax, currency)}</dd></div>
      <Separator className="my-2" />
      <div className="flex justify-between text-base font-semibold"><dt>Total</dt><dd>{formatMoney(totals.total, currency)}</dd></div>
    </dl>
  )
}
