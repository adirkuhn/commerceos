import { Fragment, useState } from 'react'
import { AlertTriangle, ChevronDown, ChevronRight } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import * as api from '../oms/api'
import type { SalesOrder, SalesOrderLineItem } from '../oms/types'

const PAGE_SIZE = 20

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

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
  const { token } = useAuth()
  const [page, setPage] = useState(1)
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
      setDetailError(err instanceof Error ? err.message : 'Failed to load line items')
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
      setLastResult({ id, text: `${result.resolved} resolved${result.still_pending > 0 ? `, ${result.still_pending} still pending` : ''}` })
      setLineItemsById(prev => { const next = { ...prev }; delete next[id]; return next })
      await refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to retry stock decrement')
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
      setLastResult({ id, text: `${result.resolved} force-accepted` })
      setLineItemsById(prev => { const next = { ...prev }; delete next[id]; return next })
      await refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to force-accept stock')
    } finally {
      setBusyId(null)
      setBusyAction(null)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <AlertTriangle size={14} className="text-amber-500" />
        <h1 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Stock Issues</h1>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{total} order{total === 1 ? '' : 's'} awaiting stock</span>
        {(error || actionError || detailError) && (
          <span className="text-xs text-red-400 ml-auto">{error || actionError || detailError}</span>
        )}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-3 py-2.5 w-6" />
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider dark:text-gray-500 text-gray-400">Order</th>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider dark:text-gray-500 text-gray-400">Customer</th>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider dark:text-gray-500 text-gray-400">Status</th>
              <th className="px-3 py-2.5 w-56" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={5} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No stock issues — everything's resolved</td></tr>
            )}
            {salesOrders.map(o => (
              <Fragment key={o.id}>
                <tr className="dark:hover:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer" onClick={() => toggleExpand(o)}>
                  <td className="px-3 py-2.5 dark:text-gray-500 text-gray-400">
                    {expandedId === o.id ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                  </td>
                  <td className="px-3 py-2.5">
                    <p className="font-medium dark:text-gray-100 text-gray-900">#{o.number || o.external_order_id}</p>
                    <p className="text-[10px] dark:text-gray-500 text-gray-400">{new Date(o.occurred_at).toLocaleString()}</p>
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{customerName(o)}</td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 capitalize">{o.status}</td>
                  <td className="px-3 py-2.5 text-right" onClick={e => e.stopPropagation()}>
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleForceAccept(o.id)}
                        disabled={busyId === o.id}
                        title="Bypass the stock check -- use only when you know the item physically exists"
                        className="px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 text-[11px] font-medium hover:bg-amber-500/10 hover:text-amber-500 disabled:opacity-50 transition-colors"
                      >
                        {busyId === o.id && busyAction === 'force' ? 'Accepting…' : 'Force accept'}
                      </button>
                      <button
                        onClick={() => handleRetry(o.id)}
                        disabled={busyId === o.id}
                        className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[11px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                      >
                        {busyId === o.id && busyAction === 'retry' ? 'Retrying…' : 'Retry stock'}
                      </button>
                    </div>
                    {lastResult?.id === o.id && (
                      <p className="text-[10px] mt-1 text-gray-400">{lastResult.text}</p>
                    )}
                  </td>
                </tr>
                {expandedId === o.id && (
                  <tr className="dark:bg-[#161B26] bg-gray-50/60">
                    <td />
                    <td colSpan={4} className="px-3 py-2.5">
                      {detailLoadingId === o.id && <p className="dark:text-gray-500 text-gray-400">Loading line items…</p>}
                      {detailLoadingId !== o.id && lineItemsById[o.id] && (
                        pendingLineItems(lineItemsById[o.id]).length === 0 ? (
                          <p className="dark:text-gray-500 text-gray-400">No pending line items found.</p>
                        ) : (
                          <table className="w-full text-[11px]">
                            <thead>
                              <tr className="dark:text-gray-500 text-gray-400">
                                <th className="text-left font-medium pb-1">Item</th>
                                <th className="text-left font-medium pb-1">SKU</th>
                                <th className="text-right font-medium pb-1">Qty short</th>
                              </tr>
                            </thead>
                            <tbody>
                              {pendingLineItems(lineItemsById[o.id]).map(li => (
                                <tr key={li.id} className="dark:text-gray-300 text-gray-700">
                                  <td className="py-1">{li.name}</td>
                                  <td className="py-1 dark:text-gray-500 text-gray-400">{li.sku_code}</td>
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
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
