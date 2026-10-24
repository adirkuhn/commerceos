import { useAuth } from '../auth/AuthContext'
import { useQueryString } from '../routing/useQueryState'
import { parseDiscount } from '../pricelist/pricing'
import { DiscountField } from '../pricelist/DiscountField'
import { PriceListTab } from '../pricelist/PriceListTab'

// Price list with a what-if discount (?discount=30). Quotes are their own
// module (/quotes).
export default function PriceList() {
  const { token } = useAuth()
  const [discountRaw, setDiscount] = useQueryString('discount')

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-end gap-3 px-5 pt-3 pb-2 bg-sunken shrink-0">
        <DiscountField value={discountRaw ?? ''} onChange={setDiscount} />
      </div>
      <PriceListTab token={token ?? ''} discount={parseDiscount(discountRaw)} />
    </div>
  )
}
