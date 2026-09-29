import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Lock } from 'lucide-react'
import * as omsApi from '../oms/api'
import { usePickingQueue } from '../oms/hooks/usePackingQueue'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { lockLabel } from '../packing/lockDisplay'
import { quantityToPick } from '../oms/pickable'
import type { SalesOrderDetail } from '../oms/types'
import { orderNumber } from '../packing/orderDisplay'
import { customerName } from '../oms/salesOrderDisplay'

interface PickOrderListScreenProps {
  token: string
  orderId: string
  onDone: () => void
  onBack: () => void
}

// Scanner picking in list mode (Settings > Fulfillment): the order's items
// as a plain list, and one button to mark the whole order picked -- no
// scanning, no session. Items are scanned later, during packing. An order
// packing already has is shown view only -- no Mark as picked.
export function PickOrderListScreen({ token, orderId, onDone, onBack }: PickOrderListScreenProps) {
  const [detail, setDetail] = useState<SalesOrderDetail | null>(null)
  const [error, setError] = useState('')
  const [marking, setMarking] = useState(false)
  const queue = usePickingQueue(token)
  const { t } = useTranslation(['scanner', 'common'])
  const nameFor = useMemberNames(token)
  const entry = queue.entries.find(e => e.sales_order.id === orderId)
  const inPacking = entry?.in_packing ?? false

  const load = useCallback(async () => {
    try {
      setDetail(await omsApi.getSalesOrder(token, orderId))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.loadOrder'))
    }
  }, [token, orderId, t])

  useEffect(() => { load() }, [load])

  async function markPicked() {
    setMarking(true)
    setError('')
    try {
      await omsApi.markOrderPicked(token, orderId)
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.markPicked'))
      setMarking(false)
    }
  }

  const items = (detail?.line_items ?? []).filter(li => quantityToPick(li) > 0)

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('orderList.backToOrders')} onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('orderList.backToOrders')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-base font-semibold">{detail ? `#${orderNumber(detail.sales_order)}` : t('orderList.order')}</h1>
          {detail && <p className="text-xs text-gray-400 truncate">{customerName(detail.sales_order)}</p>}
        </div>
      </div>
      {inPacking && (
        <p role="status" className="flex items-center gap-2 px-4 py-2 border-b border-night-700 bg-amber-500/10 text-sm text-amber-200 shrink-0">
          <Lock size={14} className="shrink-0" />
          {entry?.lock ? t('orderList.lockViewOnly', { lock: lockLabel(entry.lock, nameFor(entry.lock.user_id)) }) : t('orderList.inPackingViewOnly')}
        </p>
      )}
      <div className="flex-1 overflow-y-auto px-4 py-3">
        {!detail && !error && <p className="text-center text-sm text-gray-400 py-8">{t('common:status.loading')}</p>}
        {detail && (
          <ul aria-label={t('orderList.itemsToPick')} className="space-y-2">
            {items.map(li => (
              <li key={li.id} className="flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{li.name}</p>
                  <p className="text-xs text-gray-400 font-mono">{[li.sku_code, li.barcode].filter(Boolean).join(' · ') || t('orderList.noSku')}</p>
                </div>
                <span className="text-base font-semibold shrink-0">× {quantityToPick(li)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
      {!inPacking && (
        <div className="px-4 pb-6 pt-2 shrink-0 space-y-2">
          {error && <p role="alert" className="text-sm text-red-400">{error}</p>}
          <button
            onClick={markPicked}
            disabled={!detail || marking}
            className="w-full min-h-11 py-3 rounded-xl bg-accent-solid text-white text-sm font-semibold disabled:opacity-50"
          >
            {marking ? t('orderList.marking') : t('orderList.markPicked')}
          </button>
        </div>
      )}
    </div>
  )
}
