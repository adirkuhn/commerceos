import { useState } from 'react'
import { ArrowLeft, ShoppingCart } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import { useSalesOrderDetail } from '../oms/hooks/useSalesOrderDetail'
import type { SalesOrder, SalesOrderStatus } from '../oms/types'

const PAGE_SIZE = 20

const STATUS_COLORS: Record<SalesOrderStatus, string> = {
  pending: 'bg-gray-500/15 text-gray-400',
  processing: 'bg-blue-500/15 text-blue-400',
  'on-hold': 'bg-amber-500/15 text-amber-400',
  completed: 'bg-green-500/15 text-green-400',
  cancelled: 'bg-gray-500/15 text-gray-400',
  refunded: 'bg-purple-500/15 text-purple-400',
  failed: 'bg-red-500/15 text-red-400',
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

export default function SalesOrders() {
  const { token } = useAuth()
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const { salesOrders, total, loading, error } = useSalesOrders(token ?? '', { page, perPage: PAGE_SIZE })

  if (selectedId !== null) {
    return <SalesOrderDetailPage salesOrderId={selectedId} onClose={() => setSelectedId(null)} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{total} sales order{total === 1 ? '' : 's'}</span>
        {error && <span className="text-xs text-red-400 ml-auto">{error}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Order</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Customer</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Source</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 text-right font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Total</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No sales orders yet</td></tr>
            )}
            {salesOrders.map(o => (
              <tr
                key={o.id}
                onClick={() => setSelectedId(o.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <ShoppingCart size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">#{o.number || o.external_order_id}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">{new Date(o.occurred_at).toLocaleString()}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 max-w-[220px] truncate">{customerName(o)}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{o.source}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium', STATUS_COLORS[o.status])}>
                    {o.status}
                  </span>
                </td>
                <td className="px-3 py-2.5 text-right font-medium dark:text-gray-100 text-gray-900">
                  {formatMoney(o.total_minor_units, o.currency_code)}
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

// ── Sales Order Detail Page ──────────────────────────────────────────────────

function SalesOrderDetailPage({ salesOrderId, onClose }: { salesOrderId: string; onClose: () => void }) {
  const { token } = useAuth()
  const { detail, loading, error } = useSalesOrderDetail(token ?? '', salesOrderId)

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales Orders
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">
            {detail ? `#${detail.sales_order.number || detail.sales_order.external_order_id}` : 'Sales Order'}
          </h1>
        </div>
        {detail && (
          <span className={clsx('px-2.5 py-1 rounded-full text-xs font-medium', STATUS_COLORS[detail.sales_order.status])}>
            {detail.sales_order.status}
          </span>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
          {error && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}

          {detail && (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-1">
                  <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900 mb-2">Billing</h3>
                  <AddressLines address={detail.sales_order.billing} />
                </div>
                <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-1">
                  <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900 mb-2">Shipping</h3>
                  <AddressLines address={detail.sales_order.shipping} />
                </div>
              </div>

              <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Line items</h3>
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b dark:border-[#2D3748] border-gray-200">
                      <th className="text-left py-1.5 font-medium dark:text-gray-500 text-gray-400">Product</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400">Qty</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400">Price</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400">Total</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                    {detail.line_items.map(li => (
                      <tr key={li.id}>
                        <td className="py-1.5">
                          <p className="dark:text-gray-200 text-gray-800">{li.name}</p>
                          <p className="text-[10px] dark:text-gray-500 text-gray-400">{li.sku_code}</p>
                        </td>
                        <td className="py-1.5 text-right dark:text-gray-300 text-gray-700">{li.quantity}</td>
                        <td className="py-1.5 text-right dark:text-gray-300 text-gray-700">
                          {formatMoney(li.price_minor_units, detail.sales_order.currency_code)}
                        </td>
                        <td className="py-1.5 text-right font-medium dark:text-gray-100 text-gray-900">
                          {formatMoney(li.total_minor_units, detail.sales_order.currency_code)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>

                <div className="pt-2 border-t dark:border-[#2D3748] border-gray-200 space-y-1 text-xs">
                  <div className="flex justify-between dark:text-gray-400 text-gray-500">
                    <span>Discount</span>
                    <span>{formatMoney(detail.sales_order.discount_total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                  <div className="flex justify-between dark:text-gray-400 text-gray-500">
                    <span>Shipping</span>
                    <span>{formatMoney(detail.sales_order.shipping_total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                  <div className="flex justify-between dark:text-gray-400 text-gray-500">
                    <span>Tax</span>
                    <span>{formatMoney(detail.sales_order.tax_total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                  <div className="flex justify-between font-semibold dark:text-gray-100 text-gray-900 pt-1">
                    <span>Total</span>
                    <span>{formatMoney(detail.sales_order.total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                </div>
              </div>

              {(detail.sales_order.payment_method_title || detail.sales_order.customer_note) && (
                <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-1 text-xs">
                  {detail.sales_order.payment_method_title && (
                    <p className="dark:text-gray-400 text-gray-500">Payment: <span className="dark:text-gray-200 text-gray-800">{detail.sales_order.payment_method_title}</span></p>
                  )}
                  {detail.sales_order.customer_note && (
                    <p className="dark:text-gray-400 text-gray-500">Note: <span className="dark:text-gray-200 text-gray-800">{detail.sales_order.customer_note}</span></p>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function AddressLines({ address }: { address?: { first_name?: string; last_name?: string; company?: string; address_1?: string; address_2?: string; city?: string; state?: string; postcode?: string; country?: string; email?: string; phone?: string } }) {
  if (!address) return <p className="text-xs dark:text-gray-500 text-gray-400">—</p>
  const name = [address.first_name, address.last_name].filter(Boolean).join(' ')
  const lines = [
    name,
    address.company,
    address.address_1,
    address.address_2,
    [address.city, address.state, address.postcode].filter(Boolean).join(', '),
    address.country,
    address.email,
    address.phone,
  ].filter(Boolean)

  if (lines.length === 0) return <p className="text-xs dark:text-gray-500 text-gray-400">—</p>

  return (
    <>
      {lines.map((line, i) => (
        <p key={i} className="text-xs dark:text-gray-300 text-gray-600">{line}</p>
      ))}
    </>
  )
}
