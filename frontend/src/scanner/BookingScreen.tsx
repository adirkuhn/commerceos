import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, ChevronDown, ChevronUp, Camera, Plus, Minus, Warehouse as WarehouseIcon, X, PackagePlus, Undo2, AlertTriangle, Package, Trash2 } from 'lucide-react'
import * as api from './api'
import * as procurementApi from '../procurement/api'
import * as inventoryApi from '../inventory/api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import type { BookingDetail, BookingScan, CompleteBookingResult, MissingBookingItem, ScanBookingItemInput, BookingScanActionType } from './types'
import type { PurchaseOrder } from '../procurement/types'
import type { Warehouse, Batch } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'

// Small square thumbnail for a booking line-item row -- falls back to a
// placeholder icon when the line isn't linked to a product yet, or the
// product has no media. Reuses the same media hook/data ProductCandidateRow
// already relies on for the scan-fallback picker.
function LineItemThumb({ token, productId }: { token: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0 overflow-hidden">
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={16} className="text-gray-400" />
      )}
    </div>
  )
}

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

// A damaged/lost scan's quantity is always 0 (the write-off amount lives in
// write_off_quantity instead -- see BookingScan's doc comment), but it
// still counts toward this line's progress: it arrived, it just isn't
// sellable. Mirrors the backend's missingBookingItems fix exactly -- omitting
// write_off_quantity here made every damaged/lost unit look permanently
// missing, even once nothing actually was.
function scannedQuantityBySKU(scans: BookingDetail['scans']): Map<string, number> {
  const bySKU = new Map<string, number>()
  for (const s of scans) {
    bySKU.set(s.sku_id, (bySKU.get(s.sku_id) ?? 0) + s.quantity + s.write_off_quantity)
  }
  return bySKU
}

// Mirrors PurchaseOrderLineItem.EffectiveUnitsPerPurchasedItem -- almost
// always 1, set above 1 when the supplier's line is a box/case that isn't
// itself a catalog SKU (see ScanBookingItemUseCase's scaleByLineItem).
function effectiveUnitsPerPurchasedItem(li: { units_per_purchased_item?: number }): number {
  return li.units_per_purchased_item && li.units_per_purchased_item > 0 ? li.units_per_purchased_item : 1
}

// scans[].quantity is always stock units (ScanBookingItemUseCase scales a
// tap-to-book/RawSKU-matched entry up by EffectiveUnitsPerPurchasedItem
// before staging it -- see its own doc comment) -- this converts a SKU's
// summed stock-unit total back to purchased-item/box units, the same
// convention PurchaseOrderLineItem.Quantity itself uses, for display
// against it.
function scannedPurchasedItems(li: { units_per_purchased_item?: number }, scannedStockQty: number): number {
  return scannedStockQty / effectiveUnitsPerPurchasedItem(li)
}

// Mirrors PurchaseOrderLineItem.EffectiveUnitsPerPurchasedItem +
// CompleteBookingUseCase's missing-items math exactly, computed from data
// already loaded client-side -- lets Finish show a preview before actually
// completing the booking, no extra backend call needed.
function computeMissingItems(detail: BookingDetail): MissingBookingItem[] {
  const scannedBySKU = scannedQuantityBySKU(detail.scans)
  const missing: MissingBookingItem[] = []
  for (const li of detail.line_items) {
    const scannedQty = li.sku_id ? scannedBySKU.get(li.sku_id) ?? 0 : 0
    const expectedQty = li.quantity * effectiveUnitsPerPurchasedItem(li)
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
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [cameraOpen, setCameraOpen] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  const [result, setResult] = useState<CompleteBookingResult | null>(null)
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])
  const [pendingBookFields, setPendingBookFields] = useState<ScanBookingItemInput | null>(null)
  const [bookingLine, setBookingLine] = useState<{ skuId: string; description: string; remaining: number; scannedSoFar: number; expected: number; suggestedLotNumber?: string; suggestedExpiryDate?: string | null } | null>(null)
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

  // The actual write -- always called with sku_id already known (either
  // from BookLineSheet's line-item tap, or from resolveAndOpenSheet's code
  // resolution below), never with a raw code. Resolution and booking are
  // deliberately two separate calls so the scanner always sees a confirm
  // screen (BookLineSheet) before anything is staged, instead of the old
  // scan-and-instantly-book behavior.
  async function bookItem(fields: ScanBookingItemInput) {
    setError('')
    setScanning(true)
    try {
      await api.scanBookingItem(token, bookingId, fields)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Scan failed')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  // Builds BookLineSheet's props for a resolved SKU -- same shape a line
  // item row's onClick below builds, reused here so scan and tap land on
  // the identical confirm screen. When the SKU matches one of this
  // booking's own PO line items, mirrors that row's defaults exactly
  // (defaults to the FULL remaining quantity -- "book all" is the common
  // case, and the sheet's qty stepper is right there to change it before
  // Confirm). When it doesn't match any line item (e.g. a substitute
  // product), there's no "expected" total to default from -- falls back to
  // the resolved product name, defaulting qty to unitsPerScan (1 for a
  // direct match, or a case/box alias's multiplier).
  function openBookingLineForSKU(skuId: string, productNameFallback: string, unitsPerScan: number, opts: { lotNumber?: string; expiryDate?: string | null }, weightGrams?: number | null) {
    const scannedQty = (detail?.scans ?? []).filter(s => s.sku_id === skuId).reduce((sum, s) => sum + s.quantity, 0)
    const li = detail?.line_items.find(l => l.sku_id === skuId)
    if (li) {
      // li.quantity is purchased-item/box units -- scannedQty is stock
      // units (see scannedPurchasedItems) -- convert before comparing so
      // "expected"/"scannedSoFar"/"remaining" are all the same unit as the
      // PO line itself, matching what ScanBookingItemUseCase now scales a
      // typed quantity by on Confirm.
      const scannedBoxes = scannedPurchasedItems(li, scannedQty)
      setBookingLine({
        skuId, description: li.raw_description, expected: li.quantity, scannedSoFar: scannedBoxes,
        remaining: Math.max(li.quantity - scannedBoxes, 0),
        suggestedLotNumber: opts.lotNumber || li.suggested_lot_number, suggestedExpiryDate: opts.expiryDate ?? li.suggested_expiry_date,
      })
      return
    }
    setBookingLine({
      skuId, description: productNameFallback, expected: scannedQty, scannedSoFar: scannedQty,
      // A weight-embedded scan's decoded weight (in kg) is the real
      // quantity being received -- takes priority over unitsPerScan (1
      // for a direct match, or a case/box alias's multiplier), which
      // means nothing for a per_weight SKU.
      remaining: weightGrams != null ? weightGrams / 1000 : unitsPerScan,
      suggestedLotNumber: opts.lotNumber, suggestedExpiryDate: opts.expiryDate,
    })
  }

  // Resolves a scanned/typed code to a SKU (never books it -- see
  // resolveBookingScanCode's own doc comment), then opens the same confirm
  // screen tapping a line item row does. A camera misread or a hand-typed
  // code can be truncated/mistyped -- on an exact-match miss, falls back to
  // a partial (substring) match on product name, SKU code, or barcode (see
  // searchSKUCandidates), same as before. A single match opens the sheet
  // directly; more than one shows a picker first.
  async function resolveAndOpenSheet(code: string, opts: { lotNumber?: string; expiryDate?: string | null }) {
    setError('')
    setScanning(true)
    try {
      const resolved = await api.resolveBookingScanCode(token, bookingId, code)
      setCode('')
      openBookingLineForSKU(resolved.sku_id, resolved.product_name, resolved.units_per_scan, opts, resolved.weight_grams)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      try {
        const found = await searchSKUCandidates(token, code)
        if (found.length === 1) {
          setCode('')
          openBookingLineForSKU(found[0].sku.id, found[0].productName, 1, opts)
          return
        }
        if (found.length > 1) {
          setCandidates(found)
          setPendingBookFields({ code, quantity: 1, lot_number: opts.lotNumber, expiry_date: opts.expiryDate ?? null })
          return
        }
      } catch {
        // fall through to the original resolve error below
      }
      setError(err instanceof Error ? err.message : 'Scan failed')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  function handleCandidateBook(c: SKUCandidate) {
    if (!pendingBookFields) return
    // A name/code text match carries no real weight -- same rule
    // ScanBookingItemUseCase enforces server-side for a barcode/SKU-code
    // scan, applied here too so this fuzzy-search fallback can't bypass it.
    if (c.sku.pricing_mode === 'per_weight') {
      setCandidates([])
      setError('Priced by weight — scan its scale barcode, not a name/code match')
      return
    }
    const fields = pendingBookFields
    setCandidates([])
    setPendingBookFields(null)
    setCode('')
    openBookingLineForSKU(c.sku.id, c.productName, 1, { lotNumber: fields.lot_number, expiryDate: fields.expiry_date ?? null })
  }

  function handleScan(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    resolveAndOpenSheet(code.trim(), { lotNumber: lotNumber || undefined, expiryDate: expiryDate || null })
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    resolveAndOpenSheet(text, { lotNumber: lotNumber || undefined, expiryDate: expiryDate || null })
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

  // Undoes a mis-scan (e.g. a damaged/lost entry logged in error) --
  // reloads afterward so the line's progress and the sheet's own scan list
  // (rendered live from `detail.scans`, see BookLineSheet's `scans` prop
  // below) both drop it immediately.
  async function handleDeleteScan(scanId: string) {
    try {
      await api.deleteBookingScan(token, bookingId, scanId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to delete scan')
    }
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
      <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
          <button onClick={() => setConfirmingFinish(false)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to booking">
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
                <div key={m.line_item_id} className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                  <p className="text-sm text-gray-200">{m.raw_description}</p>
                  <p className="text-xs text-gray-400">expected {m.expected_qty} · scanned {m.scanned_qty}</p>
                </div>
              ))}
            </>
          )}
        </div>
        <div className="p-4 border-t border-night-700 shrink-0 space-y-2">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={handleConfirmComplete}
            disabled={finishing}
            className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
          >
            {finishing ? 'Completing…' : 'Confirm & Complete'}
          </button>
          <button onClick={() => setConfirmingFinish(false)} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
            Back, scan more
          </button>
        </div>
      </div>
    )
  }

  if (result) {
    return (
      <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
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
              <div key={m.line_item_id} className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <p className="text-sm text-gray-200">{m.raw_description}</p>
                <p className="text-xs text-gray-400">expected {m.expected_qty} · scanned {m.scanned_qty}</p>
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

  const scannedBySKU = scannedQuantityBySKU(detail?.scans ?? [])
  const damagedLostBySKU = new Map<string, number>()
  for (const s of detail?.scans ?? []) {
    if (s.write_off_quantity <= 0) continue
    damagedLostBySKU.set(s.sku_id, (damagedLostBySKU.get(s.sku_id) ?? 0) + s.write_off_quantity)
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button onClick={handlePause} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Booking</h1>
        <button
          onClick={() => setConfirmingFinish(true)}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700"
        >
          Finish
        </button>
      </div>

      {purchaseOrder && (
        <div className="flex items-center gap-2 px-4 py-2 border-b border-night-700 shrink-0 text-xs text-gray-400">
          <WarehouseIcon size={13} className="shrink-0" />
          <span className="flex-1 truncate">{warehouses.find(w => w.id === purchaseOrder.warehouse_id)?.name ?? 'No warehouse'}</span>
          <button type="button" onClick={() => setWarehouseSheetOpen(true)} className="text-blue-400 font-medium shrink-0">
            Change
          </button>
        </div>
      )}

      {cameraOpen && (
        <div className="px-4 pt-3 shrink-0">
          <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} frameShape="rect" variant="inline" />
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-400 py-8">Loading…</p>}
        {detail?.line_items.map(li => {
          const scannedQty = li.sku_id ? scannedBySKU.get(li.sku_id) ?? 0 : 0
          const damagedLostQty = li.sku_id ? damagedLostBySKU.get(li.sku_id) ?? 0 : 0
          // done/the row's count compare in the same units li.quantity is
          // already in (purchased items/boxes) -- scannedQty itself stays
          // in stock units for the >= check (exact, no rounding), only the
          // displayed number is converted.
          const done = scannedQty >= li.quantity * effectiveUnitsPerPurchasedItem(li)
          const scannedBoxes = scannedPurchasedItems(li, scannedQty)
          const skuId = li.sku_id
          const content = (
            <>
              <LineItemThumb token={token} productId={li.product_id} />
              {done && <Check size={16} className="text-green-400 shrink-0" />}
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate">{li.raw_description}</p>
                {!skuId && <p className="text-xs text-amber-400">Not linked to a product yet</p>}
                {damagedLostQty > 0 && <p className="text-xs text-red-400">{damagedLostQty} damaged/lost</p>}
              </div>
              <p className={`text-sm font-medium shrink-0 ${done ? 'text-green-400' : 'text-gray-300'}`}>
                {scannedBoxes} / {li.quantity}
              </p>
            </>
          )
          if (!skuId) {
            return (
              <div key={li.id} className="flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3">
                {content}
              </div>
            )
          }
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setBookingLine({
                skuId, description: li.raw_description, remaining: Math.max(li.quantity - scannedBoxes, 0), scannedSoFar: scannedBoxes, expected: li.quantity,
                suggestedLotNumber: li.suggested_lot_number, suggestedExpiryDate: li.suggested_expiry_date,
              })}
              className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-blue-500 transition-colors"
            >
              {content}
            </button>
          )
        })}
      </div>

      <form onSubmit={handleScan} className="border-t border-night-700 p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
        {candidates.length > 0 && (
          <div className="space-y-1.5">
            <p className="text-xs text-gray-400">Multiple matches -- pick one:</p>
            {candidates.map(c => (
              <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => handleCandidateBook(c)} />
            ))}
          </div>
        )}
        <div className="flex gap-2">
          <input
            ref={codeInputRef}
            value={code}
            onChange={e => setCode(e.target.value)}
            placeholder="Scan or type barcode…"
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="button"
            onClick={() => setCameraOpen(true)}
            title="Use camera"
            className="w-11 shrink-0 rounded-xl bg-night-800 border border-night-700 flex items-center justify-center text-gray-300"
          >
            <Camera size={16} />
          </button>
        </div>
        <button type="button" onClick={() => setAdvancedOpen(o => !o)} className="flex items-center gap-1 text-xs text-gray-400">
          {advancedOpen ? <ChevronUp size={12} /> : <ChevronDown size={12} />} Lot / expiry
        </button>
        {advancedOpen && (
          <div className="grid grid-cols-2 gap-2">
            <input
              value={lotNumber} onChange={e => setLotNumber(e.target.value)} placeholder="Lot number"
              className="px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            />
            <input
              type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)}
              className="px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            />
          </div>
        )}
        <button
          type="submit" disabled={scanning || !code.trim()}
          className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50"
        >
          {scanning ? 'Looking up…' : 'Scan'}
        </button>
      </form>

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
          suggestedLotNumber={bookingLine.suggestedLotNumber}
          suggestedExpiryDate={bookingLine.suggestedExpiryDate}
          scans={(detail?.scans ?? []).filter(s => s.sku_id === bookingLine.skuId)}
          onDeleteScan={handleDeleteScan}
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

// One-line label for a staged scan in BookLineSheet's history list --
// action_type plus its net effect (Quantity for receive/correction,
// WriteOffQuantity for damaged/lost, see BookingScan's own doc comment).
function scanSummary(s: BookingScan): string {
  switch (s.action_type) {
    case 'receive': return `+${s.quantity} received`
    case 'correction': return `${s.quantity} correction`
    case 'damaged': return `${s.write_off_quantity} damaged`
    case 'lost': return `${s.write_off_quantity} lost`
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
function BookLineSheet({ token, skuId, description, remaining, scannedSoFar, expected, warehouseId, pendingBatches, suggestedLotNumber, suggestedExpiryDate, scans, onDeleteScan, onCancel, onBook }: {
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
  const activeTab = LINE_SHEET_TABS.find(t => t.id === mode)!

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{description}</h2>
            <p className="text-xs text-gray-400 mt-0.5">Received {scannedSoFar} of {expected}</p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        {scans.length > 0 && (
          <div className="px-5 pt-3 space-y-1.5">
            <p className="text-xs text-gray-400">Already staged for this line:</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center gap-2 rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-200">{scanSummary(s)}</p>
                  {(s.lot_number || s.reason) && (
                    <p className="text-xs text-gray-400 truncate">{[s.lot_number, s.reason].filter(Boolean).join(' · ')}</p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title="Delete this staged scan"
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
            {LINE_SHEET_TABS.map(t => (
              <button
                key={t.id}
                type="button"
                onClick={() => switchMode(t.id)}
                className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${
                  mode === t.id ? 'bg-blue-600 text-white' : 'bg-night-800 text-gray-400'
                }`}
              >
                <t.icon size={15} />
                {t.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-gray-400 -mt-2">{activeTab.help}</p>

          {mode === 'damaged-lost' && (
            <div className="flex gap-2">
              {(['damaged', 'lost'] as const).map(t => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setDlType(t)}
                  className={`flex-1 py-2.5 rounded-xl text-sm font-medium capitalize ${dlType === t ? 'bg-red-600 text-white' : 'bg-night-800 text-gray-400'}`}
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
              <button type="button" onClick={() => setQty(String(Math.max(n - 1, 0)))} className="w-9 h-9 shrink-0 rounded-xl bg-night-800 flex items-center justify-center" aria-label="Decrease quantity">
                <Minus size={14} />
              </button>
              <input
                id="line-quantity"
                inputMode="numeric"
                value={qty}
                onChange={e => setQty(e.target.value.replace(/\D/g, ''))}
                className="flex-1 min-w-0 px-2 py-2 text-center text-base rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button type="button" onClick={() => setQty(String(n + 1))} className="w-9 h-9 shrink-0 rounded-xl bg-night-800 flex items-center justify-center" aria-label="Increase quantity">
                <Plus size={14} />
              </button>
            </div>
            {maxQty !== undefined && <p className="mt-1 text-xs text-gray-400">Up to {scannedSoFar} already received</p>}
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
              {suggestedLotNumber && (
                <p className="col-span-2 text-2xs text-blue-400">Suggested from the uploaded document — confirm to assign.</p>
              )}
              <div>
                <label htmlFor="line-lot" className="block text-xs text-gray-400 mb-1.5">Lot number</label>
                <input
                  id="line-lot"
                  value={lotNumber}
                  onChange={e => setLotNumber(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                />
              </div>
              <div>
                <label htmlFor="line-expiry" className="block text-xs text-gray-400 mb-1.5">Expiry date</label>
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
              <label htmlFor="line-reason" className="block text-xs text-gray-400 mb-1.5">Reason (optional)</label>
              <input
                id="line-reason"
                value={reason}
                onChange={e => setReason(e.target.value)}
                placeholder={mode === 'remove' ? 'e.g. counted wrong' : 'e.g. arrived crushed'}
                className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-blue-500 placeholder:text-gray-400"
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
    </Dialog>
  )
}

function BatchOption({ selected, onClick, title, subtitle }: { selected: boolean; onClick: () => void; title: string; subtitle: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full flex items-center gap-2 p-2.5 rounded-lg border text-left transition-colors ${
        selected ? 'border-blue-500 bg-blue-500/10' : 'border-night-700 bg-night-800'
      }`}
    >
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{title}</p>
        <p className="text-xs text-gray-400 truncate">{subtitle}</p>
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
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 p-5 space-y-3">
        <h2 className="text-sm font-semibold">Change warehouse</h2>
        <div className="space-y-2">
          {warehouses.map(w => (
            <button
              key={w.id}
              type="button"
              disabled={changing || w.id === currentWarehouseId}
              onClick={() => onSelect(w.id)}
              className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-blue-500 disabled:opacity-50 transition-colors"
            >
              <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
                <WarehouseIcon size={16} className="text-gray-400" />
              </div>
              <p className="text-sm font-medium flex-1 truncate">{w.name}</p>
              {w.id === currentWarehouseId && <span className="text-xs text-gray-400 shrink-0">Current</span>}
            </button>
          ))}
        </div>
        <button onClick={onCancel} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
          Cancel
        </button>
      </div>
    </Dialog>
  )
}
