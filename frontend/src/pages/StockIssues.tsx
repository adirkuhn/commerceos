import { Fragment, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, ChevronDown, ChevronRight } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import * as api from '../oms/api'
import type { SalesOrder, SalesOrderLineItem } from '../oms/types'
import { useQueryPage } from '../routing/useQueryState'
import { customerName } from '../oms/salesOrderDisplay'
import { useDateFormat } from '../format/DateFormatProvider'
import { OrderDifferencesSection } from '../oms/OrderDifferencesSection'

const PAGE_SIZE = 20

function pendingLineItems(items: SalesOrderLineItem[]): SalesOrderLineItem[] {
  return items.filter(li => li.sku_id && li.stock_status === 'pending')
}

// StockIssues lists WooCommerce sales orders where at least one linked
// (sku_id set) line item's decrement hit insufficient stock -- see
// domain/oms.SalesOrder.NeedsStockReview's doc. Each row expands to show
// exactly which line item(s) are the problem (name/sku/quantity short),
// resolved either the normal way -- staff fix the underlying stock (a
// receiving, a correction) elsewhere, then Retry here -- or, when staff
// already know the item physically exists and the tracked ledger is just
// wrong, Force accept bypasses the check entirely (see
// application/oms.ForceAcceptStockUseCase). Both deliberately manual, not
// auto-triggered by any stock fix itself.
export default function StockIssues() {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['stockIssues', 'common'])
  const { token } = useAuth()
  // In the URL (?page=) so refresh and Back keep it.
  const [page, setPage] = useQueryPage()
  const { salesOrders, total, loading, error, refresh } = useSalesOrders(token ?? '', {
    page, perPage: PAGE_SIZE, needsStockReview: true,
  })
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [lineItemsById, setLineItemsById] = useState<Record<string, SalesOrderLineItem[]>>({})
  const [detailLoadingId, setDetailLoadingId] = useState<string | null>(null)
  const [detailError, setDetailError] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [busyAction, setBusyAction] = useState<'retry' | 'force' | null>(null)
  const [actionError, setActionError] = useState('')
  const [lastResult, setLastResult] = useState<{ id: string; text: string } | null>(null)

  async function toggleExpand(order: SalesOrder) {
    if (expandedId === order.id) {
      setExpandedId(null)
      return
    }
    setExpandedId(order.id)
    if (lineItemsById[order.id]) return
    setDetailLoadingId(order.id)
    setDetailError('')
    try {
      const detail = await api.getSalesOrder(token ?? '', order.id)
      setLineItemsById(prev => ({ ...prev, [order.id]: detail.line_items }))
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : t('lineItemsFailed'))
    } finally {
      setDetailLoadingId(null)
    }
  }

  async function handleRetry(id: string) {
    setBusyId(id)
    setBusyAction('retry')
    setActionError('')
    setLastResult(null)
    try {
      const result = await api.retryStockDecrement(token ?? '', id)
      setLastResult({ id, text: result.still_pending > 0 ? t('resolvedPending', { n: result.resolved, pending: result.still_pending }) : t('resolved', { n: result.resolved }) })
      setLineItemsById(prev => { const next = { ...prev }; delete next[id]; return next })
      await refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : t('retryFailed'))
    } finally {
      setBusyId(null)
      setBusyAction(null)
    }
  }

  async function handleForceAccept(id: string) {
    setBusyId(id)
    setBusyAction('force')
    setActionError('')
    setLastResult(null)
    try {
      const result = await api.forceAcceptStock(token ?? '', id)
      setLastResult({ id, text: t('forceAccepted', { n: result.resolved }) })
      setLineItemsById(prev => { const next = { ...prev }; delete next[id]; return next })
      await refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : t('forceFailed'))
    } finally {
      setBusyId(null)
      setBusyAction(null)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <AlertTriangle size={14} className="text-warning" />
        <h1 className="text-sm font-semibold text-ink">{t('title')}</h1>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{t('awaiting', { count: total })}</span>
        {(error || actionError || detailError) && (
          <span className="text-xs text-danger ml-auto">{error || actionError || detailError}</span>
        )}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 w-6" />
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider text-ink-4">{t('colOrder')}</th>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider text-ink-4">{t('colCustomer')}</th>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider text-ink-4">{t('colStatus')}</th>
              <th className="px-3 py-2.5 w-56" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">{t('empty')}</td></tr>
            )}
            {salesOrders.map(o => (
              <Fragment key={o.id}>
                <tr className="hover:bg-hover/60 transition-colors cursor-pointer" onClick={() => toggleExpand(o)}>
                  <td className="px-3 py-2.5 text-ink-4">
                    {expandedId === o.id ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                  </td>
                  <td className="px-3 py-2.5">
                    <p className="font-medium text-ink">#{o.number || o.external_order_id}</p>
                    <p className="text-2xs text-ink-4">{formatDateTime(o.occurred_at)}</p>
                  </td>
                  <td className="px-3 py-2.5 text-ink-3">{customerName(o)}</td>
                  <td className="px-3 py-2.5 text-ink-3 capitalize">{o.status}</td>
                  <td className="px-3 py-2.5 text-right" onClick={e => e.stopPropagation()}>
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleForceAccept(o.id)}
                        disabled={busyId === o.id}
                        title={t('forceTitle')}
                        className="px-3 py-1.5 rounded-lg border border-line text-ink-2 text-2xs font-medium hover:bg-amber-500/10 hover:text-warning disabled:opacity-50 transition-colors"
                      >
                        {busyId === o.id && busyAction === 'force' ? t('accepting') : t('forceAccept')}
                      </button>
                      <button
                        onClick={() => handleRetry(o.id)}
                        disabled={busyId === o.id}
                        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
                      >
                        {busyId === o.id && busyAction === 'retry' ? t('retrying') : t('retry')}
                      </button>
                    </div>
                    {lastResult?.id === o.id && (
                      <p className="text-2xs mt-1 text-ink-4">{lastResult.text}</p>
                    )}
                  </td>
                </tr>
                {expandedId === o.id && (
                  <tr className="dark:bg-night-900 bg-gray-50/60">
                    <td />
                    <td colSpan={4} className="px-3 py-2.5">
                      {detailLoadingId === o.id && <p className="text-ink-4">{t('loadingLines')}</p>}
                      {detailLoadingId !== o.id && lineItemsById[o.id] && (
                        pendingLineItems(lineItemsById[o.id]).length === 0 ? (
                          <p className="text-ink-4">{t('noPending')}</p>
                        ) : (
                          <table className="w-full text-2xs">
                            <thead>
                              <tr className="text-ink-4">
                                <th className="text-left font-medium pb-1">{t('colItem')}</th>
                                <th className="text-left font-medium pb-1">{t('colSku')}</th>
                                <th className="text-right font-medium pb-1">{t('colShort')}</th>
                              </tr>
                            </thead>
                            <tbody>
                              {pendingLineItems(lineItemsById[o.id]).map(li => (
                                <tr key={li.id} className="text-ink-2">
                                  <td className="py-1">{li.name}</td>
                                  <td className="py-1 text-ink-4">{li.sku_code}</td>
                                  <td className="py-1 text-right">{li.quantity}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )
                      )}
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
        <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />

        <OrderDifferencesSection token={token ?? ''} />
      </div>
    </div>
  )
}
