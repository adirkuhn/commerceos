import { useEffect, useState } from 'react'
import { Search, LogOut, Package, ArrowLeft, Warehouse as WarehouseIcon } from 'lucide-react'
import * as procurementApi from '../procurement/api'
import * as suppliersApi from '../suppliers/api'
import * as inventoryApi from '../inventory/api'
import * as api from './api'
import type { PurchaseOrder } from '../procurement/types'
import type { Supplier } from '../suppliers/types'
import type { Warehouse } from '../inventory/types'

interface PurchaseOrderPickerScreenProps {
  token: string
  onBookingStarted: (bookingId: string) => void
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

export function PurchaseOrderPickerScreen({ token, onBookingStarted, onLogout, onBack }: PurchaseOrderPickerScreenProps) {
  const [purchaseOrders, setPurchaseOrders] = useState<PurchaseOrder[]>([])
  const [suppliers, setSuppliers] = useState<Supplier[]>([])
  const [warehouses, setWarehouses] = useState<Warehouse[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [startingId, setStartingId] = useState<string | null>(null)
  // Set when a selected PO has no warehouse yet -- the list is replaced by
  // an inline "pick a warehouse" step instead of navigating away, so
  // starting the booking stays a single flow.
  const [pendingPO, setPendingPO] = useState<PurchaseOrder | null>(null)

  useEffect(() => {
    Promise.all([procurementApi.listPurchaseOrders(token), suppliersApi.listSuppliers(token), inventoryApi.listWarehouses(token)])
      .then(([pos, sups, whs]) => {
        setPurchaseOrders(pos.filter(po => po.status === 'sent' || po.status === 'partially_received'))
        setSuppliers(sups)
        setWarehouses(whs)
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load purchase orders'))
      .finally(() => setLoading(false))
  }, [token])

  const supplierName = (id?: string | null) => suppliers.find(s => s.id === id)?.name ?? 'Unknown supplier'

  const filtered = purchaseOrders.filter(po =>
    query.trim() === '' || supplierName(po.supplier_id).toLowerCase().includes(query.toLowerCase()),
  )

  async function startBooking(po: PurchaseOrder, warehouseId?: string) {
    setError('')
    setStartingId(po.id)
    try {
      const booking = await api.startOrResumeBooking(token, po.id, warehouseId)
      onBookingStarted(booking.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start booking')
      setStartingId(null)
    }
  }

  function handleSelect(po: PurchaseOrder) {
    if (!po.warehouse_id) {
      setPendingPO(po)
      return
    }
    startBooking(po)
  }

  if (pendingPO) {
    return (
      <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
          <button onClick={() => setPendingPO(null)} className="p-1 -ml-1" title="Back">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="text-base font-semibold">Pick a warehouse</h1>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          <p className="text-xs text-gray-500">
            {supplierName(pendingPO.supplier_id)} has no warehouse assigned yet -- pick one to start receiving.
          </p>
          {error && <p className="text-sm text-red-400">{error}</p>}
          {warehouses.map(w => (
            <button
              key={w.id}
              onClick={() => startBooking(pendingPO, w.id)}
              disabled={startingId !== null}
              className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-left hover:border-blue-500 disabled:opacity-50 transition-colors"
            >
              <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
                <WarehouseIcon size={16} className="text-gray-400" />
              </div>
              <p className="text-sm font-medium flex-1 truncate">{w.name}</p>
              {startingId === pendingPO.id && <span className="text-xs text-blue-400 shrink-0">Starting…</span>}
            </button>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onBack} className="p-1 -ml-1" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Select a purchase order</h1>
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
            placeholder="Search by supplier…"
            className="w-full pl-9 pr-3 py-2.5 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>
      {error && <p className="mx-4 mb-2 text-sm text-red-400">{error}</p>}
      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {!loading && filtered.length === 0 && (
          <p className="text-center text-sm text-gray-500 py-8">No purchase orders ready for receiving</p>
        )}
        {filtered.map(po => (
          <button
            key={po.id}
            onClick={() => handleSelect(po)}
            disabled={startingId !== null}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-left hover:border-blue-500 disabled:opacity-50 transition-colors"
          >
            <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
              <Package size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{supplierName(po.supplier_id)}</p>
              <p className="text-xs text-gray-500 capitalize">{po.status.replace('_', ' ')}</p>
              <p className="text-xs text-gray-500 flex items-center gap-1.5">
                <span>#{po.id.slice(0, 8).toUpperCase()}</span>
                <span>·</span>
                <span>{formatDate(po.expected_date ?? po.created_at)}</span>
              </p>
            </div>
            <p className="text-xs font-medium text-gray-300 shrink-0">{formatPrice(po.total_minor_units, po.currency_code)}</p>
            {startingId === po.id && <span className="text-xs text-blue-400 shrink-0">Starting…</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
