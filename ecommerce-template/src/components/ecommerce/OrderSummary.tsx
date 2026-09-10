import type { Order } from '@/types'
import { formatMoney } from '@/lib/utils'
import { CheckoutSummary } from './CheckoutSummary'
import { Separator } from '@/components/ui/Separator'

// Order line items + the same totals breakdown CheckoutSummary shows
// during checkout -- used on order confirmation and order-detail pages,
// where a real Order (not the live cart) is the source of truth.
export function OrderSummary({ order }: { order: Order }) {
  return (
    <div className="space-y-4">
      <ul className="divide-y divide-border">
        {order.lines.map((line, i) => (
          <li key={`${line.productId}-${i}`} className="flex gap-3 py-3">
            {line.imageUrl && <img src={line.imageUrl} alt="" className="h-14 w-14 shrink-0 rounded-[var(--radius)] object-cover" />}
            <div className="flex-1">
              <p className="text-sm font-medium">{line.name}</p>
              {line.variantLabel && <p className="text-xs text-muted-foreground">{line.variantLabel}</p>}
              <p className="text-xs text-muted-foreground">Qty {line.quantity}</p>
            </div>
            <p className="text-sm font-medium">{formatMoney(line.price * line.quantity, order.currency)}</p>
          </li>
        ))}
      </ul>
      <Separator />
      <CheckoutSummary
        totals={{
          subtotal: order.subtotal, discount: order.discount, shipping: order.shipping, tax: order.tax, total: order.total,
          freeShippingThreshold: 0, freeShippingRemaining: 0,
        }}
        currency={order.currency}
      />
    </div>
  )
}
