import { useEffect, useState } from 'react'
import { Search, LogOut, ShoppingCart, ArrowLeft } from 'lucide-react'
import * as omsApi from '../oms/api'
import * as api from './api'
import type { SalesOrder } from '../oms/types'

interface SalesOrderPickerScreenProps {
  token: string
  onPickingStarted: (sessionId: string) => void
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
// picking session directly.
export function SalesOrderPickerScreen({ token, onPickingStarted, onLogout, onBack }: SalesOrderPickerScreenProps) {
  const [salesOrders, setSalesOrders] = useState<SalesOrder[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [startingId, setStartingId] = useState<string | null>(null)

  useEffect(() => {
    omsApi.listSalesOrders(token, { fulfilled: false, perPage: 100 })
      .then(result => setSalesOrders(result.salesOrders))
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load sales orders'))
      .finally(() => setLoading(false))
  }, [token])

  const filtered = salesOrders.filter(o =>
    query.trim() === ''
    || customerName(o).toLowerCase().includes(query.toLowerCase())
    || (o.number || o.external_order_id).toLowerCase().includes(query.toLowerCase()),
  )

  async function handleSelect(order: SalesOrder) {
    setError('')
    setStartingId(order.id)
    try {
      const session = await api.startOrResumePicking(token, order.id)
      onPickingStarted(session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start picking')
      setStartingId(null)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onBack} className="p-1 -ml-1" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Select an order to pick</h1>
        <button onClick={onLogout} className="p-2 rounded-lg hover:bg-[#1F2937]" title="Unpair">
          <LogOut size={16} className="text-gray-400" />
        </button>
      </div>
      <div className="px-4 py-3 shrink-0">
        <div className="relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search by customer or order number…"
            className="w-full pl-9 pr-3 py-2.5 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>
      {error && <p className="mx-4 mb-2 text-sm text-red-400">{error}</p>}
      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {!loading && filtered.length === 0 && (
          <p className="text-center text-sm text-gray-500 py-8">No unfulfilled orders ready to pick</p>
        )}
        {filtered.map(o => (
          <button
            key={o.id}
            onClick={() => handleSelect(o)}
            disabled={startingId !== null}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-left hover:border-blue-500 disabled:opacity-50 transition-colors"
          >
            <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
              <ShoppingCart size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{customerName(o)}</p>
              <p className="text-xs text-gray-500 flex items-center gap-1.5">
                <span>#{o.number || o.external_order_id}</span>
                <span>·</span>
                <span>{formatDate(o.occurred_at)}</span>
              </p>
            </div>
            <p className="text-xs font-medium text-gray-300 shrink-0">{formatPrice(o.total_minor_units, o.currency_code)}</p>
            {startingId === o.id && <span className="text-xs text-blue-400 shrink-0">Starting…</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
