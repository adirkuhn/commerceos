import { useState } from 'react'
import { PackageSearch, ChevronDown, ChevronRight } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import { useSalesOrderDetail } from '../oms/hooks/useSalesOrderDetail'
import * as api from '../oms/api'
import type { SalesOrder } from '../oms/types'
import { ProductSkuPicker } from './Procurement'
import type { SKU } from '../pms/types'
import { useQueryPage } from '../routing/useQueryState'

const PAGE_SIZE = 20

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

// UnmatchedSkus lists WooCommerce sales orders with at least one line item
// never matched to a catalog SKU (sku_id nil) -- see
// domain/oms.SalesOrder.NeedsSKUReview's doc. Staff pick the real product/
// SKU per unmatched line here; linking does not itself retry the stock
// decrement (see application/oms.LinkLineItemSKUUseCase's doc) -- resolve
// any resulting stock issue separately from the "Stock Issues" page.
export default function UnmatchedSkus() {
  const { token } = useAuth()
  // In the URL (?page=) so refresh and Back keep it.
  const [page, setPage] = useQueryPage()
  const { salesOrders, total, loading, error, refresh } = useSalesOrders(token ?? '', {
    page, perPage: PAGE_SIZE, needsSkuReview: true,
  })
  const [expandedId, setExpandedId] = useState<string | null>(null)

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <PackageSearch size={14} className="text-warning" />
        <h1 className="text-sm font-semibold text-ink">Unmatched SKUs</h1>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{total} order{total === 1 ? '' : 's'} with an unmatched line item</span>
        {error && <span className="text-xs text-danger ml-auto">{error}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        {loading && <p className="px-5 py-6 text-center text-xs text-ink-4">Loading…</p>}
        {!loading && salesOrders.length === 0 && (
          <p className="px-5 py-6 text-center text-xs text-ink-4">No unmatched SKUs — everything's linked</p>
        )}
        <div className="divide-y divide-line/60">
          {salesOrders.map(o => (
            <div key={o.id}>
              <button
                onClick={() => setExpandedId(id => (id === o.id ? null : o.id))}
                className="w-full flex items-center gap-2 px-5 py-2.5 text-left hover:bg-hover/60 transition-colors"
              >
                {expandedId === o.id ? <ChevronDown size={12} className="text-ink-4" /> : <ChevronRight size={12} className="text-ink-4" />}
                <div className="flex-1">
                  <p className="text-xs font-medium text-ink">#{o.number || o.external_order_id}</p>
                  <p className="text-2xs text-ink-4">{customerName(o)} · {new Date(o.occurred_at).toLocaleString()}</p>
                </div>
              </button>
              {expandedId === o.id && (
                <UnmatchedLineItems salesOrderId={o.id} onLinked={refresh} />
              )}
            </div>
          ))}
        </div>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}

function UnmatchedLineItems({ salesOrderId, onLinked }: { salesOrderId: string; onLinked: () => void }) {
  const { token } = useAuth()
  const { detail, loading, error, refresh } = useSalesOrderDetail(token ?? '', salesOrderId)
  const [pickerForLineItemId, setPickerForLineItemId] = useState<string | null>(null)
  const [linkingId, setLinkingId] = useState<string | null>(null)
  const [linkError, setLinkError] = useState('')

  async function handleLink(lineItemId: string, sku: SKU) {
    setLinkingId(lineItemId)
    setLinkError('')
    setPickerForLineItemId(null)
    try {
      await api.linkLineItemSku(token ?? '', salesOrderId, lineItemId, { sku_id: sku.id })
      await refresh()
      onLinked()
    } catch (err) {
      setLinkError(err instanceof Error ? err.message : 'Failed to link SKU')
    } finally {
      setLinkingId(null)
    }
  }

  const unmatched = detail?.line_items.filter(li => !li.sku_id) ?? []

  return (
    <div className="px-5 pb-3 pl-9">
      {loading && <p className="text-xs text-ink-4">Loading line items…</p>}
      {(error || linkError) && <p className="text-xs text-danger">{error || linkError}</p>}
      {!loading && unmatched.length === 0 && (
        <p className="text-xs text-ink-4">No unmatched line items remain on this order</p>
      )}
      <div className="space-y-2">
        {unmatched.map(li => (
          <div key={li.id} className="rounded-lg border border-line p-2.5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs text-ink">{li.name}</p>
                <p className="text-2xs text-ink-4">Woo SKU: {li.sku_code || '(none)'} · Qty {li.quantity}</p>
              </div>
              <button
                onClick={() => setPickerForLineItemId(id => (id === li.id ? null : li.id))}
                disabled={linkingId === li.id}
                className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors shrink-0"
              >
                {linkingId === li.id ? 'Linking…' : 'Link to product'}
              </button>
            </div>
            {pickerForLineItemId === li.id && (
              <div className="mt-2">
                <ProductSkuPicker onSelect={sku => handleLink(li.id, sku)} onClose={() => setPickerForLineItemId(null)} />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
