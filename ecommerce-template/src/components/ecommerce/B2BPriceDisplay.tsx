import { Building2 } from 'lucide-react'
import { ProductPrice } from './ProductPrice'
import { Badge } from '@/components/ui/Badge'
import { formatMoney } from '@/lib/utils'
import { applyPriceList } from '@/lib/b2bPricing'
import { PRICE_LISTS } from '@/config/demoData'
import { useAuthStore } from '@/stores/authStore'

// Price-list-aware price display: a logged-in customer whose company has
// a price list sees their real negotiated price (list price crossed out
// above it), everyone else just sees ProductPrice like normal. Reads the
// price list straight from config/demoData.ts's PRICE_LISTS -- a real
// backend would resolve this server-side from the company's own record
// instead, same seam as everything else in repositories/.
export function B2BPriceDisplay({
  price, compareAtPrice, currency = 'USD', size = 'lg', className,
}: { price: number; compareAtPrice?: number; currency?: string; size?: 'sm' | 'default' | 'lg'; className?: string }) {
  const company = useAuthStore(s => s.session?.customer.company)
  const priceList = company?.priceListId ? PRICE_LISTS.find(pl => pl.id === company.priceListId) : undefined

  if (!priceList) {
    return <ProductPrice price={price} compareAtPrice={compareAtPrice} currency={currency} size={size} className={className} />
  }

  const companyPrice = applyPriceList(price, priceList)

  return (
    <div className={className}>
      <div className="flex items-center gap-2">
        <Badge variant="secondary" className="gap-1"><Building2 className="h-3 w-3" /> {priceList.name} price</Badge>
        <span className="text-sm text-muted-foreground line-through">{formatMoney(price, currency)}</span>
      </div>
      <ProductPrice price={companyPrice} currency={currency} size={size} className="mt-1" />
    </div>
  )
}
