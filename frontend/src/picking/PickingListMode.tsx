import { useState } from 'react'
import { CheckCircle, ClipboardList, Lock } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import * as omsApi from '../oms/api'
import { quantityToPick } from '../oms/pickable'
import { useSalesOrderDetail } from '../oms/hooks/useSalesOrderDetail'
import { usePickingQueue } from '../oms/hooks/usePackingQueue'
import type { SalesOrder } from '../oms/types'
import { orderNumber } from '../packing/orderDisplay'
import { lockLabel } from '../packing/lockDisplay'
import { useQueryString } from '../routing/useQueryState'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { ReadOnlyOrderItems } from './ReadOnlyOrderItems'
import { customerName } from '../oms/salesOrderDisplay'
import { PickItemRow } from './PickItemRow'
import { apiErrorText } from '../api/apiErrorText'

// The picking module's list mode (Settings > Fulfillment): no scanning, no
// session. Pickers open an order (?order=<id>, so refresh/Back work), fetch
// its items off the shelves, and mark the whole order picked with one
// button. A picked order leaves this list and waits in the packing queue,
// where the items are scanned. An order packing already has opens as a
// read-only item list (no Mark as picked).
export function PickingListMode({ token }: { token: string }) {
  const { t } = useTranslation(['picking', 'common'])
  const { entries, loading, error: listError, refresh } = usePickingQueue(token)
  const nameFor = useMemberNames(token)
  const [openId, openOrder] = useQueryString('order', { history: 'push' })
  const [, closeOrder] = useQueryString('order')
  const openEntry = entries.find(e => e.sales_order.id === openId)
  const inPacking = openEntry?.in_packing ?? false
  const { detail, loading: detailLoading, error: detailError } = useSalesOrderDetail(token, inPacking ? null : openId)
  const [marking, setMarking] = useState(false)
  const [markError, setMarkError] = useState('')
  const [notice, setNotice] = useState('')

  function select(id: string) {
    setMarkError('')
    setNotice('')
    openOrder(id)
  }

  async function markPicked(order: SalesOrder) {
    setMarking(true)
    setMarkError('')
    try {
      await omsApi.markOrderPicked(token, order.id)
      setNotice(t('list.markedNotice', { number: orderNumber(order) }))
      closeOrder(null)
      await refresh()
    } catch (err) {
      setMarkError(apiErrorText(err) ?? t('errors.markPicked'))
    } finally {
      setMarking(false)
    }
  }

  const items = (detail?.line_items ?? []).filter(li => quantityToPick(li) > 0)

  return (
    <div className="flex h-full">
      <div className="w-64 border-r border-line bg-sunken flex flex-col overflow-hidden shrink-0">
        <div className="px-4 py-3 border-b border-line">
          <p className="text-xs font-semibold text-ink">{t('queue.title')}</p>
          <p className="text-2xs text-ink-4 mt-0.5">{t('queue.count', { count: entries.length })}</p>
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {loading && <p className="px-4 py-3 text-2xs text-ink-4">{t('common:status.loading')}</p>}
          {listError && <p className="px-4 py-3 text-2xs text-danger">{listError}</p>}
          {!loading && !listError && entries.length === 0 && <p className="px-4 py-3 text-2xs text-ink-4">{t('queue.empty')}</p>}
          {entries.map(({ sales_order: o, lock, in_packing }) => (
            <button
              key={o.id}
              aria-pressed={openId === o.id}
              onClick={() => select(o.id)}
              className={clsx(
                'w-full text-left px-4 py-3 border-b border-line transition-colors',
                openId === o.id ? 'bg-accent/10 border-l-2 border-l-accent' : 'hover:bg-raised',
              )}
            >
              <span className="block text-xs font-medium text-ink">#{orderNumber(o)}</span>
              <span className="block text-2xs text-ink-3 truncate">{customerName(o)}</span>
              {(lock || in_packing) && (
                <span className="flex items-center gap-1 mt-1 text-2xs text-ink-4">
                  <Lock size={10} className="shrink-0" />
                  <span className="truncate">{lock ? lockLabel(lock, nameFor(lock.user_id)) : t('queue.inPacking')}</span>
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {notice && (
          <p role="status" className="flex items-center gap-2 mb-4 text-xs text-success">
            <CheckCircle size={14} /> {notice}
          </p>
        )}
        {!openId && (
          <div className="flex flex-col items-center justify-center h-full text-ink-4">
            <ClipboardList size={32} className="mb-3 opacity-40" />
            <p className="text-sm">{t('list.selectOrder')}</p>
          </div>
        )}
        {openId && inPacking && openEntry && (
          <ReadOnlyOrderItems token={token} orderId={openId} lock={openEntry.lock} nameFor={nameFor} />
        )}
        {openId && !inPacking && detailLoading && !detail && <p className="text-sm text-ink-4">{t('common:status.loading')}</p>}
        {openId && !inPacking && detailError && <p role="alert" className="text-sm text-danger">{detailError}</p>}
        {openId && !inPacking && detail && (
          <div className="max-w-xl space-y-4">
            <div>
              <h2 className="text-base font-semibold text-ink">#{orderNumber(detail.sales_order)}</h2>
              <p className="text-xs text-ink-3">{customerName(detail.sales_order)}</p>
            </div>
            <ul aria-label={t('list.itemsToPick')} className="bg-raised rounded-xl border border-line divide-y divide-line">
              {items.map(li => <PickItemRow key={li.id} lineItem={li} />)}
            </ul>
            {markError && <p role="alert" className="text-xs text-danger">{markError}</p>}
            <button
              onClick={() => markPicked(detail.sales_order)}
              disabled={marking}
              className="px-4 py-2 text-sm font-medium rounded-lg bg-accent-solid text-white disabled:opacity-50"
            >
              {marking ? t('list.marking') : t('list.markPicked')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
