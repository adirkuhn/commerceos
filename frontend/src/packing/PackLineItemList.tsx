import { CheckCircle } from 'lucide-react'
import clsx from 'clsx'
import type { PickingSessionLineItem } from '../scanner/types'
import { ProductThumb, ADMIN_THUMB } from '../pms/ProductThumb'
import { SoldExpiryNote } from '../oms/SoldExpiryNote'

interface PackLineItemListProps {
  token: string
  lineItems: PickingSessionLineItem[]
  packedByLineItem: Map<string, number>
  readOnly: boolean
  // Opens the line's pack/substitute/refund panel.
  onOpenLine: (lineItem: PickingSessionLineItem, packedSoFar: number) => void
}

// The order's lines with how much of each is packed; tapping one opens it.
export function PackLineItemList({ token, lineItems, packedByLineItem, readOnly, onOpenLine }: PackLineItemListProps) {
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-2">
      {lineItems.map(li => {
        const packed = packedByLineItem.get(li.id) ?? 0
        const done = packed >= li.quantity
        const partial = packed > 0 && packed < li.quantity
        return (
          <div
            key={li.id}
            onClick={() => !readOnly && onOpenLine(li, packed)}
            className={clsx(
              'flex items-center gap-4 p-3 rounded-xl border transition-all',
              li.sku_id && !readOnly && 'cursor-pointer',
              done ? 'dark:bg-green-500/5 dark:border-green-500/20 border-green-200 bg-green-50' :
              partial ? 'dark:bg-amber-500/5 dark:border-amber-500/20 border-amber-200 bg-amber-50' :
              'bg-raised border-line',
            )}
          >
            {done ? (
              <div className="w-10 h-10 rounded-lg bg-green-500/20 flex items-center justify-center shrink-0">
                <CheckCircle size={18} className="text-success" />
              </div>
            ) : (
              <ProductThumb token={token} productId={li.product_id} appearance={ADMIN_THUMB} />
            )}
            <div className="flex-1 min-w-0">
              <p className={clsx('text-sm font-medium', done ? 'dark:text-green-300 text-success' : 'text-ink')}>{li.name}</p>
              <span className="text-2xs font-mono text-ink-4">{li.sku_code}</span>
              <SoldExpiryNote date={li.sold_expiry_date} />
            </div>
            <div className="text-right shrink-0">
              <p className={clsx('text-lg font-bold tabular-nums', done ? 'text-success' : partial ? 'text-warning' : 'text-ink')}>
                {packed}/{li.quantity}
              </p>
              <p className="text-2xs text-ink-4">packed</p>
            </div>
          </div>
        )
      })}
    </div>
  )
}
