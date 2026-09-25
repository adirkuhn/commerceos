import { Lock } from 'lucide-react'
import { useSalesOrderDetail } from '../oms/hooks/useSalesOrderDetail'
import { quantityToPick } from '../oms/pickable'
import type { QueueLock } from '../oms/types'
import { orderNumber } from '../packing/orderDisplay'
import { lockLabel } from '../packing/lockDisplay'
import { customerName } from '../oms/salesOrderDisplay'

interface ReadOnlyOrderItemsProps {
  token: string
  orderId: string
  // Who is packing it right now, if anyone.
  lock: QueueLock | null
  nameFor: (userId: string) => string
}

// Picking's view of an order packing has: just its items, no actions (no
// scan, pick or mark picked, no session) -- picking stays out of the way.
export function ReadOnlyOrderItems({ token, orderId, lock, nameFor }: ReadOnlyOrderItemsProps) {
  const { detail, loading, error } = useSalesOrderDetail(token, orderId)
  const items = (detail?.line_items ?? []).filter(li => quantityToPick(li) > 0)

  return (
    <div className="max-w-xl space-y-4">
      <p role="status" className="flex items-center gap-2 rounded-lg bg-warning/10 px-3 py-2 text-xs text-ink-2">
        <Lock size={12} className="shrink-0" />
        {lock ? `${lockLabel(lock, nameFor(lock.user_id))} -- view only` : 'This order is in packing -- view only'}
      </p>
      {loading && !detail && <p className="text-sm text-ink-4">Loading…</p>}
      {error && <p role="alert" className="text-sm text-danger">{error}</p>}
      {detail && (
        <>
          <div>
            <h2 className="text-base font-semibold text-ink">#{orderNumber(detail.sales_order)}</h2>
            <p className="text-xs text-ink-3">{customerName(detail.sales_order)}</p>
          </div>
          <ul aria-label="Items to pick" className="bg-raised rounded-xl border border-line divide-y divide-line">
            {items.map(li => (
              <li key={li.id} className="flex items-center gap-3 px-4 py-3">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-ink truncate">{li.name}</p>
                  <p className="text-2xs text-ink-4 font-mono">{[li.sku_code, li.barcode].filter(Boolean).join(' · ') || 'No SKU'}</p>
                </div>
                <span className="text-sm font-semibold text-ink shrink-0">× {quantityToPick(li)}</span>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}
