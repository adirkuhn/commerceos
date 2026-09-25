import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, ChevronDown, ChevronUp, Camera, Warehouse as WarehouseIcon } from 'lucide-react'
import * as api from './api'
import * as procurementApi from '../procurement/api'
import * as inventoryApi from '../inventory/api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import type { BookingDetail, CompleteBookingResult, ScanBookingItemInput, BookingScanActionType } from './types'
import type { PurchaseOrder } from '../procurement/types'
import type { Warehouse } from '../inventory/types'
import { BookLineSheet, WarehouseSheet } from './BookingLineSheet'
import { computeMissingItems, effectiveUnitsPerPurchasedItem, isAuthError, pendingBatchesForSKU, scannedPurchasedItems, scannedQuantityBySKU } from './bookingHelpers'
import { LineItemThumb } from './BookingLineItemThumb'

interface BookingScreenProps {
  token: string
  bookingId: string
  onExit: () => void
  onAuthError: () => void
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
          <button aria-label="Back to booking" onClick={() => setConfirmingFinish(false)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to booking">
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
            {finishing ? 'Completing…' : 'Confirm & complete'}
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
          <button onClick={onExit} className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover">
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
        <button aria-label="Pause and go back" onClick={handlePause} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Pause and go back">
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
          <button type="button" onClick={() => setWarehouseSheetOpen(true)} className="text-accent font-medium shrink-0">
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
              className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-accent transition-colors"
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
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
          <button aria-label="Use camera"
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
          className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
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
