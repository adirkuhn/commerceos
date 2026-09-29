import { useTranslation } from 'react-i18next'
import { quantityToPick } from '../oms/pickable'
import type { SalesOrderLineItem } from '../oms/types'
import { SoldExpiryNote } from '../oms/SoldExpiryNote'

// One line of list-mode picking's "Items to pick": name, SKU/barcode,
// the sold expiry Woo reported, and how many are still to pick. Shared by
// the actionable view and the read-only one an order in packing gets.
export function PickItemRow({ lineItem: li }: { lineItem: SalesOrderLineItem }) {
  const { t } = useTranslation('picking')
  return (
    <li className="flex items-center gap-3 px-4 py-3">
      <div className="flex-1 min-w-0">
        <p className="text-sm text-ink truncate">{li.name}</p>
        <p className="text-2xs text-ink-4 font-mono">{[li.sku_code, li.barcode].filter(Boolean).join(' · ') || t('item.noSku')}</p>
        <SoldExpiryNote date={li.sold_expiry_date} />
      </div>
      <span className="text-sm font-semibold text-ink shrink-0">× {quantityToPick(li)}</span>
    </li>
  )
}
