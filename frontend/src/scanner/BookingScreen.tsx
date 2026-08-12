import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, ChevronDown, ChevronUp, Camera, Plus, Minus, Warehouse as WarehouseIcon, X, PackagePlus, Undo2, AlertTriangle } from 'lucide-react'
import * as api from './api'
import * as procurementApi from '../procurement/api'
import * as inventoryApi from '../inventory/api'
import { CameraScanner } from './CameraScanner'
import type { BookingDetail, CompleteBookingResult, MissingBookingItem, ScanBookingItemInput, BookingScanActionType } from './types'
import type { PurchaseOrder } from '../procurement/types'
import type { Warehouse, Batch } from '../inventory/types'

interface BookingScreenProps {
  token: string
  bookingId: string
  onExit: () => void
  onAuthError: () => void
}

function isAuthError(err: unknown): boolean {
  return err instanceof Error && /401|unauthorized/i.test(err.message)
}

export interface PendingBatch {
  lotNumber: string
  expiryDate?: string | null
}

// A "receive" scan with a lot_number and no real batch_id is targeting a
// batch that won't exist until Finish (see CompleteBookingUseCase's replay,
// which resolves it by matching sku_id+lot_number across every staged row
// for the booking, regardless of who scanned it). Surfacing these in the
// batch picker is what lets a second scanner working the same booking add
// to a batch the first one just started, by name, instead of having to
// guess/retype the exact lot text.
function pendingBatchesForSKU(detail: BookingDetail | null, skuId: string): PendingBatch[] {
  if (!detail) return []
  const seen = new Map<string, string | null | undefined>()
  for (const s of detail.scans) {
    if (s.sku_id !== skuId || s.action_type !== 'receive' || s.batch_id || !s.lot_number) continue
    if (!seen.has(s.lot_number)) seen.set(s.lot_number, s.expiry_date)
  }
  return Array.from(seen, ([lotNumber, expiryDate]) => ({ lotNumber, expiryDate }))
}

// Mirrors PurchaseOrderLineItem.EffectiveUnitsPerPurchasedItem +
// CompleteBookingUseCase's missing-items math exactly, computed from data
// already loaded client-side -- lets Finish show a preview before actually
// completing the booking, no extra backend call needed.
function computeMissingItems(detail: BookingDetail): MissingBookingItem[] {
  const scannedBySKU = new Map<string, number>()
  for (const s of detail.scans) {
    scannedBySKU.set(s.sku_id, (scannedBySKU.get(s.sku_id) ?? 0) + s.quantity)
  }
  const missing: MissingBookingItem[] = []
  for (const li of detail.line_items) {
    const scannedQty = li.sku_id ? scannedBySKU.get(li.sku_id) ?? 0 : 0
    const effectiveUnitsPerPurchasedItem = li.units_per_purchased_item && li.units_per_purchased_item > 0 ? li.units_per_purchased_item : 1
    const expectedQty = li.quantity * effectiveUnitsPerPurchasedItem
    if (scannedQty < expectedQty) {
      missing.push({ line_item_id: li.id, raw_description: li.raw_description, expected_qty: expectedQty, scanned_qty: scannedQty })
    }
  }
  return missing
}

export function BookingScreen({ token, bookingId, onExit, onAuthError }: BookingScreenProps) {
  const [detail, setDetail] = useState<BookingDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [code, setCode] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [cameraOpen, setCameraOpen] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  const [result, setResult] = useState<CompleteBookingResult | null>(null)
  const [bookingLine, setBookingLine] = useState<{ skuId: string; description: string; remaining: number; scannedSoFar: number; expected: number } | null>(null)
  const [purchaseOrder, setPurchaseOrder] = useState<PurchaseOrder | null>(null)
  const [warehouses, setWarehouses] = useState<Warehouse[]>([])
  const [warehouseSheetOpen, setWarehouseSheetOpen] = useState(false)
  const [changingWarehouse, setChangingWarehouse] = useState(false)
  const codeInputRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    try {
      const detail = await api.getBooking(token, bookingId)
      setDetail(detail)
      const [poDetail, whs] = await Promise.all([
        procurementApi.getPurchaseOrder(token, detail.booking.purchase_order_id),
        inventoryApi.listWarehouses(token),
      ])
      setPurchaseOrder(poDetail.purchase_order)
      setWarehouses(whs)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to load booking')
    } finally {
      setLoading(false)
    }
  }

  async function handleChangeWarehouse(newWarehouseId: string) {
    if (!purchaseOrder) return
    setError('')
    setChangingWarehouse(true)
    try {
      const updated = await procurementApi.updatePurchaseOrder(token, purchaseOrder.id, {
        supplier_id: purchaseOrder.supplier_id ?? '',
        warehouse_id: newWarehouseId,
        status: purchaseOrder.status,
        currency_code: purchaseOrder.currency_code,
        expected_date: purchaseOrder.expected_date ?? null,
        notes: purchaseOrder.notes,
      })
      setPurchaseOrder(updated)
      setWarehouseSheetOpen(false)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to change warehouse')
    } finally {
      setChangingWarehouse(false)
    }
  }

  useEffect(() => { load() }, [bookingId])
  useEffect(() => { codeInputRef.current?.focus() }, [detail])

  // Every field the caller wants must be passed explicitly -- no implicit
  // merge with the outer scan-form's lotNumber/expiryDate state, which
  // would otherwise silently stomp whatever a caller like BookLineSheet
  // passes for its own batch/lot/expiry/reason.
  async function bookItem(fields: ScanBookingItemInput) {
    setError('')
    setScanning(true)
    try {
      await api.scanBookingItem(token, bookingId, fields)
      setCode('')
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Scan failed')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  function handleScan(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    bookItem({ code: code.trim(), quantity: parseInt(quantity, 10) || 1, lot_number: lotNumber || undefined, expiry_date: expiryDate || null })
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    setCode(text)
    bookItem({ code: text, quantity: parseInt(quantity, 10) || 1, lot_number: lotNumber || undefined, expiry_date: expiryDate || null })
  }

  // Receive, Remove (correction), and Damaged/Lost are all just staged
  // BookingScan rows now -- action_type tells CompleteBookingUseCase's
  // replay what to do with each one at Finish. Nothing here touches real
  // inventory directly.
  function handleRowConfirm(fields: { action_type: BookingScanActionType; quantity: number; batch_id?: string | null; lot_number?: string; expiry_date?: string | null; reason?: string }) {
    if (!bookingLine) return
    bookItem({ sku_id: bookingLine.skuId, ...fields })
    setBookingLine(null)
  }

  async function handlePause() {
    try {
      await api.pauseBooking(token, bookingId)
      onExit()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to pause')
    }
  }

  async function handleConfirmComplete() {
    setFinishing(true)
    try {
      setResult(await api.completeBooking(token, bookingId))
      setConfirmingFinish(false)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to complete')
    } finally {
      setFinishing(false)
    }
  }

  if (confirmingFinish && detail) {
    const missing = computeMissingItems(detail)
    return (
      <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
          <button onClick={() => setConfirmingFinish(false)} className="p-1 -ml-1" title="Back to booking">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          {missing.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-8">Every line is fully scanned.</p>
          ) : (
            <>
              <p className="text-sm text-amber-400">{missing.length} line(s) aren't fully scanned yet:</p>
              {missing.map(m => (
                <div key={m.line_item_id} className="rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                  <p className="text-sm text-gray-200">{m.raw_description}</p>
                  <p className="text-xs text-gray-500">expected {m.expected_qty} · scanned {m.scanned_qty}</p>
                </div>
              ))}
            </>
          )}
        </div>
        <div className="p-4 border-t border-[#2D3748] shrink-0 space-y-2">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={handleConfirmComplete}
            disabled={finishing}
            className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
          >
            {finishing ? 'Completing…' : 'Confirm & Complete'}
          </button>
          <button onClick={() => setConfirmingFinish(false)} className="w-full py-2.5 rounded-xl bg-[#1F2937] text-gray-300 text-sm font-medium">
            Back, scan more
          </button>
        </div>
      </div>
    )
  }

  if (result) {
    return (
      <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
        <div className="flex flex-col items-center text-center gap-3 px-6 pt-8 pb-4 shrink-0">
          <div className="w-14 h-14 rounded-2xl bg-green-600/20 flex items-center justify-center">
            <PackageCheck size={26} className="text-green-400" />
          </div>
          <h1 className="text-lg font-semibold">Booking complete</h1>
          {result.missing_items.length === 0 && <p className="text-sm text-gray-400">Every line was fully scanned.</p>}
        </div>
        {result.missing_items.length > 0 && (
          <div className="flex-1 overflow-y-auto px-6 pb-4 space-y-2">
            <p className="text-sm text-amber-400">{result.missing_items.length} line(s) weren't fully scanned:</p>
            {result.missing_items.map(m => (
              <div key={m.line_item_id} className="rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                <p className="text-sm text-gray-200">{m.raw_description}</p>
                <p className="text-xs text-gray-500">expected {m.expected_qty} · scanned {m.scanned_qty}</p>
              </div>
            ))}
          </div>
        )}
        <div className="p-6 shrink-0">
          <button onClick={onExit} className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700">
            Done
          </button>
        </div>
      </div>
    )
  }

  const scannedBySKU = new Map<string, number>()
  for (const s of detail?.scans ?? []) {
    scannedBySKU.set(s.sku_id, (scannedBySKU.get(s.sku_id) ?? 0) + s.quantity)
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={handlePause} className="p-1 -ml-1" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Booking</h1>
        <button
          onClick={() => setConfirmingFinish(true)}
          className="px-3 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700"
        >
          Finish
        </button>
      </div>

      {purchaseOrder && (
        <div className="flex items-center gap-2 px-4 py-2 border-b border-[#2D3748] shrink-0 text-xs text-gray-400">
          <WarehouseIcon size={13} className="shrink-0" />
          <span className="flex-1 truncate">{warehouses.find(w => w.id === purchaseOrder.warehouse_id)?.name ?? 'No warehouse'}</span>
          <button type="button" onClick={() => setWarehouseSheetOpen(true)} className="text-blue-400 font-medium shrink-0">
            Change
          </button>
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {detail?.line_items.map(li => {
          const scannedQty = li.sku_id ? scannedBySKU.get(li.sku_id) ?? 0 : 0
          const done = scannedQty >= li.quantity
          const skuId = li.sku_id
          const content = (
            <>
              {done && <Check size={16} className="text-green-400 shrink-0" />}
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate">{li.raw_description}</p>
                {!skuId && <p className="text-xs text-amber-400">Not linked to a product yet</p>}
              </div>
              <p className={`text-sm font-medium shrink-0 ${done ? 'text-green-400' : 'text-gray-300'}`}>
                {scannedQty} / {li.quantity}
              </p>
            </>
          )
          if (!skuId) {
            return (
              <div key={li.id} className="flex items-center gap-3 rounded-xl bg-[#1F2937] border border-[#2D3748] px-3.5 py-3">
                {content}
              </div>
            )
          }
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setBookingLine({ skuId, description: li.raw_description, remaining: Math.max(li.quantity - scannedQty, 0), scannedSoFar: scannedQty, expected: li.quantity })}
              className="w-full flex items-center gap-3 rounded-xl bg-[#1F2937] border border-[#2D3748] px-3.5 py-3 text-left hover:border-blue-500 transition-colors"
            >
              {content}
            </button>
          )
        })}
      </div>

      <form onSubmit={handleScan} className="border-t border-[#2D3748] p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
        <div className="flex gap-2">
          <input
            ref={codeInputRef}
            value={code}
            onChange={e => setCode(e.target.value)}
            placeholder="Scan or type barcode…"
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            type="number" min="1" value={quantity} onChange={e => setQuantity(e.target.value)}
            className="w-16 px-2 py-3 text-sm text-center rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="button"
            onClick={() => setCameraOpen(true)}
            title="Use camera"
            className="w-11 shrink-0 rounded-xl bg-[#1F2937] border border-[#2D3748] flex items-center justify-center text-gray-300"
          >
            <Camera size={16} />
          </button>
        </div>
        <button type="button" onClick={() => setAdvancedOpen(o => !o)} className="flex items-center gap-1 text-xs text-gray-500">
          {advancedOpen ? <ChevronUp size={12} /> : <ChevronDown size={12} />} Lot / expiry
        </button>
        {advancedOpen && (
          <div className="grid grid-cols-2 gap-2">
            <input
              value={lotNumber} onChange={e => setLotNumber(e.target.value)} placeholder="Lot number"
              className="px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none"
            />
            <input
              type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)}
              className="px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none"
            />
          </div>
        )}
        <button
          type="submit" disabled={scanning || !code.trim()}
          className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50"
        >
          {scanning ? 'Saving…' : 'Add'}
        </button>
      </form>

      {cameraOpen && (
        <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} />
      )}

      {bookingLine && (
        <BookLineSheet
          token={token}
          skuId={bookingLine.skuId}
          description={bookingLine.description}
          remaining={bookingLine.remaining}
          scannedSoFar={bookingLine.scannedSoFar}
          expected={bookingLine.expected}
          warehouseId={purchaseOrder?.warehouse_id ?? ''}
          pendingBatches={pendingBatchesForSKU(detail, bookingLine.skuId)}
          onCancel={() => setBookingLine(null)}
          onBook={handleRowConfirm}
        />
      )}

      {warehouseSheetOpen && purchaseOrder && (
        <WarehouseSheet
          warehouses={warehouses}
          currentWarehouseId={purchaseOrder.warehouse_id ?? null}
          changing={changingWarehouse}
          onCancel={() => setWarehouseSheetOpen(false)}
          onSelect={handleChangeWarehouse}
        />
      )}
    </div>
  )
}

// Tap-to-book: the touch-only equivalent of scanning a barcode -- tapping a
// line item row books directly against its already-known SKU (see
// ScanBookingItemInput.sku_id), prefilled with the remaining quantity.
type LineSheetMode = 'receive' | 'remove' | 'damaged-lost'

const LINE_SHEET_TABS: { id: LineSheetMode; label: string; icon: typeof PackagePlus; help: string }[] = [
  { id: 'receive', label: 'Receive', icon: PackagePlus, help: 'Add stock for this item, into an existing or a new batch.' },
  { id: 'remove', label: 'Remove', icon: Undo2, help: "Fix a mistake -- you scanned more than actually arrived. Reduces what's counted as received." },
  { id: 'damaged-lost', label: 'Damaged/Lost', icon: AlertTriangle, help: "It did arrive, but isn't sellable. Stays counted as received; written off separately." },
]

function formatBatchExpiry(iso?: string | null): string {
  if (!iso) return 'No expiry set'
  return `Expires ${new Date(iso).toLocaleDateString()}`
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
function BookLineSheet({ token, skuId, description, remaining, scannedSoFar, expected, warehouseId, pendingBatches, onCancel, onBook }: {
  token: string
  skuId: string
  description: string
  remaining: number
  scannedSoFar: number
  expected: number
  warehouseId: string
  pendingBatches: PendingBatch[]
  onCancel: () => void
  onBook: (fields: { action_type: BookingScanActionType; quantity: number; batch_id?: string | null; lot_number?: string; expiry_date?: string | null; reason?: string }) => void
}) {
  const [mode, setMode] = useState<LineSheetMode>('receive')
  const [batches, setBatches] = useState<Batch[]>([])
  // '' = general ledger, 'new' = create new (receive only), 'pending:<lot>'
  // = a not-yet-real batch staged earlier in this booking, or a real
  // batch's own id.
  const [batchId, setBatchId] = useState('')
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
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
  const activeTab = LINE_SHEET_TABS.find(t => t.id === mode)!

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{description}</h2>
            <p className="text-xs text-gray-500 mt-0.5">Received {scannedSoFar} of {expected}</p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            {LINE_SHEET_TABS.map(t => (
              <button
                key={t.id}
                type="button"
                onClick={() => switchMode(t.id)}
                className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-[11px] font-medium leading-tight ${
                  mode === t.id ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'
                }`}
              >
                <t.icon size={15} />
                {t.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-gray-500 -mt-2">{activeTab.help}</p>

          {mode === 'damaged-lost' && (
            <div className="flex gap-2">
              {(['damaged', 'lost'] as const).map(t => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setDlType(t)}
                  className={`flex-1 py-2.5 rounded-xl text-sm font-medium capitalize ${dlType === t ? 'bg-red-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
                >
                  {t}
                </button>
              ))}
            </div>
          )}

          <div>
            <label htmlFor="line-quantity" className="block text-xs text-gray-400 mb-1.5">
              {mode === 'receive' ? 'Quantity to receive' : mode === 'remove' ? 'Quantity to remove' : 'Quantity'}
            </label>
            <div className="flex items-center gap-2">
              <button type="button" onClick={() => setQty(String(Math.max(n - 1, 0)))} className="w-11 h-11 shrink-0 rounded-xl bg-[#1F2937] flex items-center justify-center" aria-label="Decrease quantity">
                <Minus size={16} />
              </button>
              <input
                id="line-quantity"
                inputMode="numeric"
                value={qty}
                onChange={e => setQty(e.target.value.replace(/\D/g, ''))}
                className="flex-1 px-3 py-2.5 text-center text-lg rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button type="button" onClick={() => setQty(String(n + 1))} className="w-11 h-11 shrink-0 rounded-xl bg-[#1F2937] flex items-center justify-center" aria-label="Increase quantity">
                <Plus size={16} />
              </button>
            </div>
            {maxQty !== undefined && <p className="mt-1 text-xs text-gray-500">Up to {scannedSoFar} already received</p>}
          </div>

          <div>
            <p className="text-xs text-gray-400 mb-1.5">{mode === 'receive' ? 'Batch' : 'From batch'}</p>
            <div className="space-y-1.5 max-h-40 overflow-y-auto">
              <BatchOption
                selected={batchId === ''}
                onClick={() => setBatchId('')}
                title="General ledger"
                subtitle="No specific batch"
              />
              {pendingBatches.map(pb => (
                <BatchOption
                  key={`${PENDING_PREFIX}${pb.lotNumber}`}
                  selected={batchId === `${PENDING_PREFIX}${pb.lotNumber}`}
                  onClick={() => setBatchId(`${PENDING_PREFIX}${pb.lotNumber}`)}
                  title={pb.lotNumber}
                  subtitle={`${formatBatchExpiry(pb.expiryDate)} · Pending -- created when the booking finishes`}
                />
              ))}
              {batches.map(b => (
                <BatchOption
                  key={b.id}
                  selected={batchId === b.id}
                  onClick={() => setBatchId(b.id)}
                  title={b.lot_number || `Batch ${b.id.slice(0, 8)}`}
                  subtitle={`${formatBatchExpiry(b.expiry_date)} · Qty ${b.quantity}`}
                />
              ))}
              {mode === 'receive' && (
                <BatchOption
                  selected={batchId === 'new'}
                  onClick={() => setBatchId('new')}
                  title="+ New batch"
                  subtitle="Start a new lot"
                />
              )}
            </div>
          </div>

          {mode === 'receive' && batchId === 'new' && (
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label htmlFor="line-lot" className="block text-xs text-gray-400 mb-1.5">Lot number</label>
                <input
                  id="line-lot"
                  value={lotNumber}
                  onChange={e => setLotNumber(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none"
                />
              </div>
              <div>
                <label htmlFor="line-expiry" className="block text-xs text-gray-400 mb-1.5">Expiry date</label>
                <input
                  id="line-expiry"
                  type="date"
                  value={expiryDate}
                  onChange={e => setExpiryDate(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none"
                />
              </div>
            </div>
          )}

          {mode !== 'receive' && (
            <div>
              <label htmlFor="line-reason" className="block text-xs text-gray-400 mb-1.5">Reason (optional)</label>
              <input
                id="line-reason"
                value={reason}
                onChange={e => setReason(e.target.value)}
                placeholder={mode === 'remove' ? 'e.g. counted wrong' : 'e.g. arrived crushed'}
                className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500 placeholder:text-gray-600"
              />
            </div>
          )}

          <button
            onClick={handleConfirm}
            disabled={confirmDisabled}
            className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${mode === 'damaged-lost' ? 'bg-red-600' : 'bg-blue-600'}`}
          >
            Confirm
          </button>
        </div>
      </div>
    </div>
  )
}

function BatchOption({ selected, onClick, title, subtitle }: { selected: boolean; onClick: () => void; title: string; subtitle: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full flex items-center gap-2 p-2.5 rounded-lg border text-left transition-colors ${
        selected ? 'border-blue-500 bg-blue-500/10' : 'border-[#2D3748] bg-[#1F2937]'
      }`}
    >
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{title}</p>
        <p className="text-xs text-gray-500 truncate">{subtitle}</p>
      </div>
      {selected && <Check size={14} className="text-blue-400 shrink-0" />}
    </button>
  )
}

// Changing warehouse only affects scans recorded from this point forward --
// ScanBookingItemUseCase reads the PO's warehouse fresh on every scan, and
// stock movements already written are immutable historical fact, so nothing
// retroactively moves.
function WarehouseSheet({ warehouses, currentWarehouseId, changing, onCancel, onSelect }: {
  warehouses: Warehouse[]
  currentWarehouseId: string | null
  changing: boolean
  onCancel: () => void
  onSelect: (warehouseId: string) => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] p-5 space-y-3">
        <h2 className="text-sm font-semibold">Change warehouse</h2>
        <div className="space-y-2">
          {warehouses.map(w => (
            <button
              key={w.id}
              type="button"
              disabled={changing || w.id === currentWarehouseId}
              onClick={() => onSelect(w.id)}
              className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-left hover:border-blue-500 disabled:opacity-50 transition-colors"
            >
              <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
                <WarehouseIcon size={16} className="text-gray-400" />
              </div>
              <p className="text-sm font-medium flex-1 truncate">{w.name}</p>
              {w.id === currentWarehouseId && <span className="text-xs text-gray-500 shrink-0">Current</span>}
            </button>
          ))}
        </div>
        <button onClick={onCancel} className="w-full py-2.5 rounded-xl bg-[#1F2937] text-gray-300 text-sm font-medium">
          Cancel
        </button>
      </div>
    </div>
  )
}
