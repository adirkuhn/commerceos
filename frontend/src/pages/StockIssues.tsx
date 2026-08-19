import { useState } from 'react'
import { AlertTriangle } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import * as api from '../oms/api'
import type { SalesOrder } from '../oms/types'

const PAGE_SIZE = 20

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

// StockIssues lists WooCommerce sales orders where at least one linked
// (sku_id set) line item's decrement hit insufficient stock -- see
// domain/oms.SalesOrder.NeedsStockReview's doc. Staff fix the underlying
// stock (a receiving, a correction) elsewhere, then click Retry here to
// re-attempt the decrement -- see application/oms.RetryStockDecrementUseCase.
// Deliberately manual, not auto-triggered by the stock fix itself.
export default function StockIssues() {
  const { token } = useAuth()
  const [page, setPage] = useState(1)
  const { salesOrders, total, loading, error, refresh } = useSalesOrders(token ?? '', {
    page, perPage: PAGE_SIZE, needsStockReview: true,
  })
  const [retryingId, setRetryingId] = useState<string | null>(null)
  const [retryError, setRetryError] = useState('')
  const [lastResult, setLastResult] = useState<{ id: string; resolved: number; stillPending: number } | null>(null)

  async function handleRetry(id: string) {
    setRetryingId(id)
    setRetryError('')
    setLastResult(null)
    try {
      const result = await api.retryStockDecrement(token ?? '', id)
      setLastResult({ id, resolved: result.resolved, stillPending: result.still_pending })
      await refresh()
    } catch (err) {
      setRetryError(err instanceof Error ? err.message : 'Failed to retry stock decrement')
    } finally {
      setRetryingId(null)
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
        {(error || retryError) && <span className="text-xs text-red-400 ml-auto">{error || retryError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider dark:text-gray-500 text-gray-400">Order</th>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider dark:text-gray-500 text-gray-400">Customer</th>
              <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider dark:text-gray-500 text-gray-400">Status</th>
              <th className="px-3 py-2.5 w-40" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={4} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={4} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No stock issues — everything's resolved</td></tr>
            )}
            {salesOrders.map(o => (
              <tr key={o.id} className="dark:hover:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                <td className="px-3 py-2.5">
                  <p className="font-medium dark:text-gray-100 text-gray-900">#{o.number || o.external_order_id}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">{new Date(o.occurred_at).toLocaleString()}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{customerName(o)}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 capitalize">{o.status}</td>
                <td className="px-3 py-2.5 text-right">
                  <button
                    onClick={() => handleRetry(o.id)}
                    disabled={retryingId === o.id}
                    className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[11px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                  >
                    {retryingId === o.id ? 'Retrying…' : 'Retry stock'}
                  </button>
                  {lastResult?.id === o.id && (
                    <p className={clsx('text-[10px] mt-1', lastResult.stillPending > 0 ? 'text-amber-500' : 'text-green-500')}>
                      {lastResult.resolved} resolved{lastResult.stillPending > 0 ? `, ${lastResult.stillPending} still pending` : ''}
                    </p>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
