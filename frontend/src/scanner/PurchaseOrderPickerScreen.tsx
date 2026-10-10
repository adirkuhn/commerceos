import { useEffect, useState } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { Search, LogOut, Package, ArrowLeft, Warehouse as WarehouseIcon } from 'lucide-react'
import * as procurementApi from '../procurement/api'
import * as suppliersApi from '../suppliers/api'
import * as inventoryApi from '../inventory/api'
import * as api from './api'
import type { PurchaseOrder } from '../procurement/types'
import type { Supplier } from '../suppliers/types'
import type { Warehouse } from '../inventory/types'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

interface PurchaseOrderPickerScreenProps {
  token: string
  onBookingStarted: (bookingId: string) => void
  onLogout: () => void
  onBack: () => void
}

// Only sent and partially received orders are listed for receiving.
function poStatusLabel(status: string): string {
  switch (status) {
    case 'sent': return i18next.t('scanner:poPicker.status.sent')
    case 'partially_received': return i18next.t('scanner:poPicker.status.partiallyReceived')
    default: return status.replace('_', ' ')
  }
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

export function PurchaseOrderPickerScreen({ token, onBookingStarted, onLogout, onBack }: PurchaseOrderPickerScreenProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['scanner', 'common'])
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
      .catch(err => setError(apiErrorText(err) ?? t('errors.loadPurchaseOrders')))
      .finally(() => setLoading(false))
  }, [token, t])

  const supplierName = (id?: string | null) => suppliers.find(s => s.id === id)?.name ?? t('poPicker.unknownSupplier')

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
      setError(apiErrorText(err) ?? t('errors.startBooking'))
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
      <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
          <button aria-label={t('shared.back')} onClick={() => setPendingPO(null)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.back')}>
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="text-base font-semibold">{t('poPicker.pickWarehouse')}</h1>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          <p className="text-xs text-gray-400">
            {t('poPicker.noWarehouse', { supplier: supplierName(pendingPO.supplier_id) })}
          </p>
          {error && <p className="text-sm text-red-400">{error}</p>}
          {warehouses.map(w => (
            <button
              key={w.id}
              onClick={() => startBooking(pendingPO, w.id)}
              disabled={startingId !== null}
              className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-accent disabled:opacity-50 transition-colors"
            >
              <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
                <WarehouseIcon size={16} className="text-gray-400" />
              </div>
              <p className="text-sm font-medium flex-1 truncate">{w.name}</p>
              {startingId === pendingPO.id && <span className="text-xs text-accent shrink-0">{t('pickDone.starting')}</span>}
            </button>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('shared.backToApps')} onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.backToApps')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">{t('poPicker.title')}</h1>
        <button aria-label={t('shared.unpair')} onClick={onLogout} className="p-2 rounded-lg hover:bg-night-800" title={t('shared.unpair')}>
          <LogOut size={16} className="text-gray-400" />
        </button>
      </div>
      <div className="px-4 py-3 shrink-0">
        <div className="relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder={t('poPicker.search')}
            className="w-full pl-9 pr-3 py-2.5 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
        </div>
      </div>
      {error && <p className="mx-4 mb-2 text-sm text-red-400">{error}</p>}
      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
        {loading && <p className="text-center text-sm text-gray-400 py-8">{t('common:status.loading')}</p>}
        {!loading && filtered.length === 0 && (
          <p className="text-center text-sm text-gray-400 py-8">{t('poPicker.empty')}</p>
        )}
        {filtered.map(po => (
          <button
            key={po.id}
            onClick={() => handleSelect(po)}
            disabled={startingId !== null}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-accent disabled:opacity-50 transition-colors"
          >
            <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
              <Package size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{supplierName(po.supplier_id)}</p>
              <p className="text-xs text-gray-400">{poStatusLabel(po.status)}</p>
              <p className="text-xs text-gray-400 flex items-center gap-1.5">
                <span>#{po.id.slice(0, 8).toUpperCase()}</span>
                <span>·</span>
                <span>{formatDate(po.expected_date ?? po.created_at)}</span>
              </p>
            </div>
            <p className="text-xs font-medium text-gray-300 shrink-0">{formatPrice(po.total_minor_units, po.currency_code)}</p>
            {startingId === po.id && <span className="text-xs text-accent shrink-0">{t('pickDone.starting')}</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
