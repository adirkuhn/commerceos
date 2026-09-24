import { useState } from 'react'
import { Search, LogOut, ShoppingCart, ArrowLeft, Lock } from 'lucide-react'
import { usePickingQueue } from '../oms/hooks/usePackingQueue'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { lockLabel } from '../packing/lockDisplay'
import { usePickingMode } from '../picking/usePickingMode'
import * as api from './api'
import type { SalesOrder } from '../oms/types'

interface SalesOrderPickerScreenProps {
  token: string
  onPickingStarted: (sessionId: string, order: SalesOrder) => void
  // List mode (Settings > Fulfillment): open the order's plain item list
  // instead of starting a scan session.
  onOrderChosenForList: (orderId: string) => void
  onLogout: () => void
  onBack: () => void
}

function formatDate(iso?: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || 'Unknown customer'
}

// Unlike PurchaseOrderPickerScreen, there's no warehouse-assignment step --
// SalesOrder.WarehouseID is always set, so selecting an order starts a
// picking session directly -- or, in list mode, opens its item list.
export function SalesOrderPickerScreen({ token, onPickingStarted, onOrderChosenForList, onLogout, onBack }: SalesOrderPickerScreenProps) {
  const { mode, error: modeError } = usePickingMode(token)
  const queue = usePickingQueue(token)
  const nameFor = useMemberNames(token)
  const [query, setQuery] = useState('')
  const [startError, setStartError] = useState('')
  const [startingId, setStartingId] = useState<string | null>(null)
  const loading = queue.loading
  const error = startError || queue.error

  const filtered = queue.entries.filter(({ sales_order: o }) =>
    query.trim() === ''
    || customerName(o).toLowerCase().includes(query.toLowerCase())
    || (o.number || o.external_order_id).toLowerCase().includes(query.toLowerCase()),
  )

  async function handleSelect(order: SalesOrder, inPacking: boolean) {
    // An order packing has opens as its read-only item list, whatever the
    // picking mode -- picking stays out of the way.
    if (mode === 'list' || inPacking) {
      onOrderChosenForList(order.id)
      return
    }
    setStartError('')
    setStartingId(order.id)
    try {
      const session = await api.startOrResumePicking(token, order.id)
      onPickingStarted(session.id, order)
    } catch (err) {
      setStartError(err instanceof Error ? err.message : 'Failed to start picking')
      setStartingId(null)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label="Back to apps" onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Select an order to pick</h1>
        <button aria-label="Unpair" onClick={onLogout} className="p-2 rounded-lg hover:bg-night-800" title="Unpair">
          <LogOut size={16} className="text-gray-400" />
        </button>
      </div>
      <div className="px-4 py-3 shrink-0">
        <div className="relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search by customer or order number…"
            className="w-full pl-9 pr-3 py-2.5 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
        </div>
      </div>
      {(error || modeError) && <p className="mx-4 mb-2 text-sm text-red-400">{error || `Couldn't load the picking mode (${modeError}) -- using scan mode.`}</p>}
      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
        {loading && <p className="text-center text-sm text-gray-400 py-8">Loading…</p>}
        {!loading && filtered.length === 0 && (
          <p className="text-center text-sm text-gray-400 py-8">No unfulfilled orders ready to pick</p>
        )}
        {filtered.map(({ sales_order: o, lock, in_packing }) => (
          <button
            key={o.id}
            onClick={() => handleSelect(o, in_packing)}
            disabled={startingId !== null || mode === null}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-accent disabled:opacity-50 transition-colors"
          >
            <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
              <ShoppingCart size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{customerName(o)}</p>
              <p className="text-xs text-gray-400 flex items-center gap-1.5">
                <span>#{o.number || o.external_order_id}</span>
                <span>·</span>
                <span>{formatDate(o.occurred_at)}</span>
              </p>
              {(lock || in_packing) && (
                <p className="text-xs text-gray-400 flex items-center gap-1 mt-0.5">
                  <Lock size={11} className="shrink-0" />
                  <span className="truncate">{lock ? lockLabel(lock, nameFor(lock.user_id)) : 'In packing'}</span>
                </p>
              )}
            </div>
            <p className="text-xs font-medium text-gray-300 shrink-0">{formatPrice(o.total_minor_units, o.currency_code)}</p>
            {startingId === o.id && <span className="text-xs text-accent shrink-0">Starting…</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
