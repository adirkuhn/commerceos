import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import { Check, Plus, Minus, Warehouse as WarehouseIcon, X, PackagePlus, Undo2, AlertTriangle, Trash2 } from 'lucide-react'
import * as inventoryApi from '../inventory/api'
import type { BookingScan, BookingScanActionType } from './types'
import type { Warehouse, Batch } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'
import { PendingBatch } from './bookingHelpers'
import type { DateInput } from '../format/date'
import { useDateFormat } from '../format/DateFormatProvider'

// Tap-to-book: the touch-only equivalent of scanning a barcode -- tapping a
// line item row books directly against its already-known SKU (see
// ScanBookingItemInput.sku_id), prefilled with the remaining quantity.
type LineSheetMode = 'receive' | 'remove' | 'damaged-lost'

function lineSheetTabs(): { id: LineSheetMode; label: string; icon: typeof PackagePlus; help: string }[] {
  return [
    { id: 'receive', label: i18next.t('scanner:booking.tab.receive'), icon: PackagePlus, help: i18next.t('scanner:booking.tab.receiveHelp') },
    { id: 'remove', label: i18next.t('scanner:booking.tab.remove'), icon: Undo2, help: i18next.t('scanner:booking.tab.removeHelp') },
    { id: 'damaged-lost', label: i18next.t('scanner:booking.tab.damagedLost'), icon: AlertTriangle, help: i18next.t('scanner:booking.tab.damagedLostHelp') },
  ]
}

function formatBatchExpiry(iso: string | null | undefined, formatDate: (value: DateInput) => string): string {
  if (!iso) return i18next.t('scanner:batch.noExpiry')
  return i18next.t('scanner:batch.expires', { date: formatDate(iso) })
}

// One-line label for a staged scan in BookLineSheet's history list --
// action_type plus its net effect (Quantity for receive/correction,
// WriteOffQuantity for damaged/lost, see BookingScan's own doc comment).
function scanSummary(s: BookingScan): string {
  switch (s.action_type) {
    case 'receive': return i18next.t('scanner:booking.scan.receive', { quantity: s.quantity })
    case 'correction': return i18next.t('scanner:booking.scan.correction', { quantity: s.quantity })
    case 'damaged': return i18next.t('scanner:booking.scan.damaged', { quantity: s.write_off_quantity })
    case 'lost': return i18next.t('scanner:booking.scan.lost', { quantity: s.write_off_quantity })
  }
}

const PENDING_PREFIX = 'pending:'

// Everything you can do to one line item's stock while booking: receive
// (full or partial, into an existing, pending, or brand-new batch), remove
// (correct an over-/mis-scan -- reduces the booking's own scanned count),
// or mark damaged/lost (a write-off -- the stock did arrive, so it stays
// counted as received; it just isn't sellable anymore). All three are
// staged BookingScan rows now (see ScanBookingItemInput.action_type) --
// nothing here calls real inventory directly. Never increases stock:
// receive is the only mode that adds, the other two only ever remove.
export function BookLineSheet({ token, skuId, description, remaining, scannedSoFar, expected, warehouseId, pendingBatches, suggestedLotNumber, suggestedExpiryDate, scans, onDeleteScan, onCancel, onBook }: {
  token: string
  skuId: string
  description: string
  remaining: number
  scannedSoFar: number
  expected: number
  warehouseId: string
  pendingBatches: PendingBatch[]
  // Parsed from the source document (see PurchaseOrderLineItem.suggested_lot_number/
  // suggested_expiry_date) -- pre-fills the "+ New batch" lot/expiry inputs
  // below so the scanner isn't retyping what the invoice already printed,
  // but nothing is created until Confirm is tapped.
  suggestedLotNumber?: string
  suggestedExpiryDate?: string | null
  // Every staged scan for this SKU, oldest first -- rendered as a small
  // history list (action, quantity/write-off, batch, who/when) with a
  // delete action per row. Passed live from BookingScreen's `detail` state
  // rather than snapshotted, so a delete's refresh is reflected here too.
  scans: BookingScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onBook: (fields: { action_type: BookingScanActionType; quantity: number; batch_id?: string | null; lot_number?: string; expiry_date?: string | null; reason?: string }) => void
}) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['scanner', 'common'])
  const tabs = lineSheetTabs()
  const [mode, setMode] = useState<LineSheetMode>('receive')
  const [batches, setBatches] = useState<Batch[]>([])
  // '' = general ledger, 'new' = create new (receive only), 'pending:<lot>'
  // = a not-yet-real batch staged earlier in this booking, or a real
  // batch's own id. Defaults straight to 'new' with the lot/expiry inputs
  // pre-filled when the document suggested one -- still just a default,
  // the scanner can pick general ledger/pending/an existing batch instead.
  const [batchId, setBatchId] = useState(suggestedLotNumber ? 'new' : '')
  const [lotNumber, setLotNumber] = useState(suggestedLotNumber ?? '')
  const [expiryDate, setExpiryDate] = useState(suggestedExpiryDate ?? '')
  const [qty, setQty] = useState(String(remaining))
  const [reason, setReason] = useState('')
  const [dlType, setDlType] = useState<'damaged' | 'lost'>('damaged')
  const n = Number(qty) || 0

  useEffect(() => {
    inventoryApi.listBatches(token, skuId).then(all => setBatches(all.filter(b => b.warehouse_id === warehouseId && !b.is_general_ledger)))
  }, [token, skuId, warehouseId])

  function switchMode(next: LineSheetMode) {
    setMode(next)
    setQty(next === 'receive' ? String(remaining) : '')
    setBatchId('')
  }

  function handleConfirm() {
    const actionType: BookingScanActionType = mode === 'receive' ? 'receive' : mode === 'remove' ? 'correction' : dlType
    const isPending = batchId.startsWith(PENDING_PREFIX)
    onBook({
      action_type: actionType,
      quantity: n,
      batch_id: batchId && batchId !== 'new' && !isPending ? batchId : undefined,
      lot_number: batchId === 'new' ? lotNumber : isPending ? batchId.slice(PENDING_PREFIX.length) : undefined,
      expiry_date: batchId === 'new' ? (expiryDate || null) : undefined,
      reason: mode === 'receive' ? undefined : (reason || undefined),
    })
  }

  const maxQty = mode === 'receive' ? undefined : scannedSoFar
  const confirmDisabled = n <= 0 || (maxQty !== undefined && n > maxQty)
  const activeTab = tabs.find(tab => tab.id === mode)!

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{description}</h2>
            <p className="text-xs text-gray-400 mt-0.5">{t('booking.receivedOf', { scanned: scannedSoFar, expected })}</p>
          </div>
          <button aria-label={t('common:actions.close')} onClick={onCancel} title={t('common:actions.close')} className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        {scans.length > 0 && (
          <div className="px-5 pt-3 space-y-1.5">
            <p className="text-xs text-gray-400">{t('shared.alreadyStaged')}</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center gap-2 rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-200">{scanSummary(s)}</p>
                  {(s.lot_number || s.reason) && (
                    <p className="text-xs text-gray-400 truncate">{[s.lot_number, s.reason].filter(Boolean).join(' · ')}</p>
                  )}
                </div>
                <button aria-label={t('shared.deleteStagedScan')}
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title={t('shared.deleteStagedScan')}
                  className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            {tabs.map(tab => (
              <button aria-pressed={mode === tab.id}
                key={tab.id}
                type="button"
                onClick={() => switchMode(tab.id)}
                className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${
                  mode === tab.id ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'
                }`}
              >
                <tab.icon size={15} />
                {tab.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-gray-400 -mt-2">{activeTab.help}</p>

          {mode === 'damaged-lost' && (
            <div className="flex gap-2">
              {(['damaged', 'lost'] as const).map(kind => (
                <button aria-pressed={dlType === kind}
                  key={kind}
                  type="button"
                  onClick={() => setDlType(kind)}
                  className={`flex-1 py-2.5 rounded-xl text-sm font-medium ${dlType === kind ? 'bg-red-600 text-white' : 'bg-night-800 text-gray-400'}`}
                >
                  {kind === 'damaged' ? t('booking.damaged') : t('booking.lost')}
                </button>
              ))}
            </div>
          )}

          <div>
            <label htmlFor="line-quantity" className="block text-xs text-gray-400 mb-1.5">
              {mode === 'receive' ? t('booking.qtyReceive') : mode === 'remove' ? t('booking.qtyRemove') : t('shared.quantity')}
            </label>
            <div className="flex items-center gap-2">
              <button type="button" onClick={() => setQty(String(Math.max(n - 1, 0)))} className="w-11 h-11 shrink-0 rounded-xl bg-night-800 flex items-center justify-center" aria-label={t('shared.decreaseQuantity')}>
                <Minus size={14} />
              </button>
              <input
                id="line-quantity"
                inputMode="numeric"
                value={qty}
                onChange={e => setQty(e.target.value.replace(/\D/g, ''))}
                className="flex-1 min-w-0 px-2 py-2 text-center text-base rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
              />
              <button type="button" onClick={() => setQty(String(n + 1))} className="w-11 h-11 shrink-0 rounded-xl bg-night-800 flex items-center justify-center" aria-label={t('shared.increaseQuantity')}>
                <Plus size={14} />
              </button>
            </div>
            {maxQty !== undefined && <p className="mt-1 text-xs text-gray-400">{t('booking.upTo', { quantity: scannedSoFar })}</p>}
          </div>

          <div>
            <p className="text-xs text-gray-400 mb-1.5">{mode === 'receive' ? t('booking.batch') : t('booking.fromBatch')}</p>
            <div className="space-y-1.5 max-h-40 overflow-y-auto">
              <BatchOption
                selected={batchId === ''}
                onClick={() => setBatchId('')}
                title={t('batch.generalLedger')}
                subtitle={t('batch.noSpecific')}
              />
              {pendingBatches.map(pb => (
                <BatchOption
                  key={`${PENDING_PREFIX}${pb.lotNumber}`}
                  selected={batchId === `${PENDING_PREFIX}${pb.lotNumber}`}
                  onClick={() => setBatchId(`${PENDING_PREFIX}${pb.lotNumber}`)}
                  title={pb.lotNumber}
                  subtitle={t('booking.pendingBatch', { expiry: formatBatchExpiry(pb.expiryDate, formatDate) })}
                />
              ))}
              {batches.map(b => (
                <BatchOption
                  key={b.id}
                  selected={batchId === b.id}
                  onClick={() => setBatchId(b.id)}
                  title={b.lot_number || t('batch.fallbackName', { id: b.id.slice(0, 8) })}
                  subtitle={t('batch.expiryQty', { expiry: formatBatchExpiry(b.expiry_date, formatDate), quantity: b.quantity })}
                />
              ))}
              {mode === 'receive' && (
                <BatchOption
                  selected={batchId === 'new'}
                  onClick={() => setBatchId('new')}
                  title={t('batch.newBatch')}
                  subtitle={t('batch.startNewLot')}
                />
              )}
            </div>
          </div>

          {mode === 'receive' && batchId === 'new' && (
            <div className="grid grid-cols-2 gap-2">
              {suggestedLotNumber && (
                <p className="col-span-2 text-2xs text-accent">{t('booking.suggested')}</p>
              )}
              <div>
                <label htmlFor="line-lot" className="block text-xs text-gray-400 mb-1.5">{t('batch.lotNumber')}</label>
                <input
                  id="line-lot"
                  value={lotNumber}
                  onChange={e => setLotNumber(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                />
              </div>
              <div>
                <label htmlFor="line-expiry" className="block text-xs text-gray-400 mb-1.5">{t('batch.expiryDate')}</label>
                <input
                  id="line-expiry"
                  type="date"
                  value={expiryDate}
                  onChange={e => setExpiryDate(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                />
              </div>
            </div>
          )}

          {mode !== 'receive' && (
            <div>
              <label htmlFor="line-reason" className="block text-xs text-gray-400 mb-1.5">{t('shared.reasonOptional')}</label>
              <input
                id="line-reason"
                value={reason}
                onChange={e => setReason(e.target.value)}
                placeholder={mode === 'remove' ? t('booking.reasonRemoveExample') : t('booking.reasonDamagedExample')}
                className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus placeholder:text-gray-400"
              />
            </div>
          )}

          <button
            onClick={handleConfirm}
            disabled={confirmDisabled}
            className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${mode === 'damaged-lost' ? 'bg-red-600' : 'bg-accent-solid'}`}
          >
            {t('shared.confirm')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}

function BatchOption({ selected, onClick, title, subtitle }: { selected: boolean; onClick: () => void; title: string; subtitle: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={`w-full flex items-center gap-2 p-2.5 rounded-lg border text-left transition-colors ${
        selected ? 'border-accent bg-accent/10' : 'border-night-700 bg-night-800'
      }`}
    >
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{title}</p>
        <p className="text-xs text-gray-400 truncate">{subtitle}</p>
      </div>
      {selected && <Check size={14} className="text-accent shrink-0" />}
    </button>
  )
}

// Changing warehouse only affects scans recorded from this point forward --
// ScanBookingItemUseCase reads the PO's warehouse fresh on every scan, and
// stock movements already written are immutable historical fact, so nothing
// retroactively moves.
export function WarehouseSheet({ warehouses, currentWarehouseId, changing, onCancel, onSelect }: {
  warehouses: Warehouse[]
  currentWarehouseId: string | null
  changing: boolean
  onCancel: () => void
  onSelect: (warehouseId: string) => void
}) {
  const { t } = useTranslation(['scanner', 'common'])
  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 p-5 space-y-3">
        <h2 className="text-sm font-semibold">{t('booking.changeWarehouse')}</h2>
        <div className="space-y-2">
          {warehouses.map(w => (
            <button
              key={w.id}
              type="button"
              disabled={changing || w.id === currentWarehouseId}
              onClick={() => onSelect(w.id)}
              className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-accent disabled:opacity-50 transition-colors"
            >
              <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
                <WarehouseIcon size={16} className="text-gray-400" />
              </div>
              <p className="text-sm font-medium flex-1 truncate">{w.name}</p>
              {w.id === currentWarehouseId && <span className="text-xs text-gray-400 shrink-0">{t('booking.current')}</span>}
            </button>
          ))}
        </div>
        <button onClick={onCancel} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
          {t('common:actions.cancel')}
        </button>
      </div>
    </Dialog>
  )
}
