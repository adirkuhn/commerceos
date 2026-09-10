import type { QuantityDiscountTier } from '@/types'
import { formatMoney } from '@/lib/utils'
import { priceForQuantity } from '@/lib/b2bPricing'

// Shown on the product page for any product carrying wholesale terms
// (see config/demoData.ts's WHOLESALE_TERMS) -- most of the catalog has
// none, in which case ProductDetailPage never renders this at all.
export function QuantityBreaksTable({
  basePrice, currency, minOrderQuantity, tiers,
}: { basePrice: number; currency: string; minOrderQuantity?: number; tiers?: QuantityDiscountTier[] }) {
  if (!tiers || tiers.length === 0) return null
  const sorted = [...tiers].sort((a, b) => a.minQuantity - b.minQuantity)

  return (
    <div className="rounded-[var(--radius)] border border-border p-4">
      <p className="mb-1 text-sm font-medium">Wholesale pricing</p>
      {minOrderQuantity && <p className="mb-3 text-xs text-muted-foreground">Minimum order quantity: {minOrderQuantity} units</p>}
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="text-xs uppercase tracking-wide text-muted-foreground">
            <th className="pb-1.5 font-medium">Quantity</th>
            <th className="pb-1.5 font-medium">Discount</th>
            <th className="pb-1.5 text-right font-medium">Unit price</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {sorted.map(tier => (
            <tr key={tier.minQuantity}>
              <td className="py-1.5">{tier.minQuantity}+ units</td>
              <td className="py-1.5">-{tier.discountPercent}%</td>
              <td className="py-1.5 text-right font-medium">{formatMoney(priceForQuantity(basePrice, tier.minQuantity, tiers), currency)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
