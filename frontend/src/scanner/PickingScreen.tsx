import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, Camera, X, Package, Repeat, Gift, PackagePlus, Search } from 'lucide-react'
import * as api from './api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { QuantityStepper } from './QuantityStepper'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import { playErrorSound } from './errorSound'
import type {
  PickingSessionDetail, PickingSessionLineItem, PickScan, CompletePickingResult,
  MissingPickItem, PickScanActionType,
} from './types'
import type { SalesOrder, Address } from '../oms/types'

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || 'Unknown customer'
}

function shippingAddressLine(address?: Address): string {
  if (!address) return ''
  const streetAndLocality = [address.address_1, address.address_2, [address.postcode, address.city].filter(Boolean).join(' ')]
    .filter(Boolean).join(', ')
  return [streetAndLocality, address.country].filter(Boolean).join(', ')
}

// Falls back to a placeholder icon when the line isn't linked to a product
// yet, or the product has no media -- same convention as BookingScreen's
// own LineItemThumb for procurement bookings.
function LineItemThumb({ token, name, productId }: { token: string; name: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0 overflow-hidden" title={name}>
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={16} className="text-gray-400" />
      )}
    </div>
  )
}

interface PickingScreenProps {
  token: string
  sessionId: string
  order: SalesOrder
  onExit: () => void
  onAuthError: () => void
}

function isAuthError(err: unknown): boolean {
  return err instanceof Error && /401|unauthorized/i.test(err.message)
}

// A pick or substitute scan's net effect on a line's picked progress --
// mirrors backend missingPickItems' scoring exactly (freebie/new_item never
// count against an original line, they have no line_item_id).
function pickedQuantityByLineItem(scans: PickScan[]): Map<string, number> {
  const byLineItem = new Map<string, number>()
  for (const s of scans) {
    if (!s.line_item_id) continue
    if (s.action_type !== 'pick' && s.action_type !== 'substitute') continue
    byLineItem.set(s.line_item_id, (byLineItem.get(s.line_item_id) ?? 0) + s.quantity)
  }
  return byLineItem
}

// Mirrors CompletePickingUseCase's missingPickItems exactly, computed
// client-side from data already loaded -- lets Finish show a preview before
// actually completing the session.
function computeMissingItems(detail: PickingSessionDetail): MissingPickItem[] {
  const pickedByLineItem = pickedQuantityByLineItem(detail.scans)
  const missing: MissingPickItem[] = []
  for (const li of detail.line_items) {
    const picked = pickedByLineItem.get(li.id) ?? 0
    if (picked < li.quantity) {
      missing.push({ line_item_id: li.id, name: li.name, expected_qty: li.quantity, picked_qty: picked })
    }
  }
  return missing
}

export function PickingScreen({ token, sessionId, order, onExit, onAuthError }: PickingScreenProps) {
  const [detail, setDetail] = useState<PickingSessionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [code, setCode] = useState('')
  const [scanning, setScanning] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)
  const [cameraOpen, setCameraOpen] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  const [result, setResult] = useState<CompletePickingResult | null>(null)
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])
  const [pickLine, setPickLine] = useState<{ lineItem: PickingSessionLineItem; pickedSoFar: number } | null>(null)
  const [addItemOpen, setAddItemOpen] = useState(false)
  const codeInputRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    try {
      setDetail(await api.getPickingSession(token, sessionId))
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to load picking session')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [sessionId])
  useEffect(() => { codeInputRef.current?.focus() }, [detail])

  async function stage(input: Parameters<typeof api.scanPickItem>[2]) {
    setError('')
    setScanning(true)
    try {
      await api.scanPickItem(token, sessionId, input)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Scan failed')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  // A scanned/typed code matching one of this order's own line items marks
  // one unit picked immediately -- no confirm sheet, just an increment (the
  // sheet is reserved for a deliberate tap on the line itself, to pick a
  // specific quantity or substitute -- see setPickLine's row-click callers
  // below). Already-fully-picked (over-scan) or resolved-but-not-on-this-
  // order both block further scanning behind scanErrorAlert until the
  // picker explicitly confirms the error -- mis-scans must never silently
  // fall through to the freebie/new-item flow. A resolve miss still falls
  // back to a partial name/code/barcode search (same as Booking); only once
  // that search also comes up empty does it become a blocking error too.
  function routeResolvedSku(skuId: string) {
    const li = detail?.line_items.find(l => l.sku_id === skuId)
    if (li) {
      const pickedSoFar = pickedQuantityByLineItem(detail?.scans ?? []).get(li.id) ?? 0
      if (pickedSoFar >= li.quantity) {
        playErrorSound()
        setScanErrorAlert('Already fully picked')
        return
      }
      stage({ line_item_id: li.id, sku_id: skuId, action_type: 'pick', quantity: 1 })
    } else {
      playErrorSound()
      setScanErrorAlert('Item not on this order')
    }
  }

  async function resolveAndRoute(scannedCode: string) {
    if (scanErrorAlert) return
    setError('')
    setScanning(true)
    try {
      const resolved = await api.resolvePickScanCode(token, sessionId, scannedCode)
      setCode('')
      routeResolvedSku(resolved.sku_id)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      try {
        const found = await searchSKUCandidates(token, scannedCode)
        if (found.length > 0) {
          setCode('')
          setCandidates(found)
          return
        }
      } catch {
        // fall through to the blocking error below
      }
      playErrorSound()
      setScanErrorAlert('Barcode not recognized')
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  function handleCandidateSelect(c: SKUCandidate) {
    setCandidates([])
    setCode('')
    routeResolvedSku(c.sku.id)
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeInputRef.current?.focus()
  }

  function handleScan(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    resolveAndRoute(code.trim())
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    resolveAndRoute(text)
  }

  async function handleDeleteScan(scanId: string) {
    try {
      await api.deletePickScan(token, sessionId, scanId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to delete scan')
    }
  }

  async function handlePause() {
    try {
      await api.pausePicking(token, sessionId)
      onExit()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to pause')
    }
  }

  async function handleConfirmComplete() {
    setFinishing(true)
    try {
      setResult(await api.completePicking(token, sessionId))
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
          <button onClick={() => setConfirmingFinish(false)} className="p-1 -ml-1" title="Back to picking">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          {missing.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-8">Every line is fully picked.</p>
          ) : (
            <>
              <p className="text-sm text-amber-400">{missing.length} line(s) aren't fully picked yet:</p>
              {missing.map(m => (
                <div key={m.line_item_id} className="rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                  <p className="text-sm text-gray-200">{m.name}</p>
                  <p className="text-xs text-gray-500">expected {m.expected_qty} · picked {m.picked_qty}</p>
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
          <h1 className="text-lg font-semibold">Order picked</h1>
          {result.missing_items.length === 0 && <p className="text-sm text-gray-400">Every line was fully picked.</p>}
        </div>
        {result.missing_items.length > 0 && (
          <div className="flex-1 overflow-y-auto px-6 pb-4 space-y-2">
            <p className="text-sm text-amber-400">{result.missing_items.length} line(s) weren't fully picked:</p>
            {result.missing_items.map(m => (
              <div key={m.line_item_id} className="rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                <p className="text-sm text-gray-200">{m.name}</p>
                <p className="text-xs text-gray-500">expected {m.expected_qty} · picked {m.picked_qty}</p>
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

  const pickedByLineItem = pickedQuantityByLineItem(detail?.scans ?? [])
  const addedItems = (detail?.scans ?? []).filter(s => s.action_type === 'freebie' || s.action_type === 'new_item')

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={handlePause} className="p-1 -ml-1" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5">
            <h1 className="text-sm font-semibold truncate">#{order.number || order.external_order_id}</h1>
            <span className="text-xs text-gray-500">·</span>
            <span className="text-xs text-gray-400 truncate">{customerName(order)}</span>
          </div>
          {shippingAddressLine(order.shipping) && (
            <p className="text-[11px] text-gray-500 truncate">{shippingAddressLine(order.shipping)}</p>
          )}
        </div>
        <button
          onClick={() => setConfirmingFinish(true)}
          className="px-3 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700 shrink-0"
        >
          Finish
        </button>
      </div>

      {cameraOpen && (
        <div className="px-4 pt-3 shrink-0">
          <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} frameShape="rect" variant="inline" />
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {detail?.line_items.map(li => {
          const pickedQty = pickedByLineItem.get(li.id) ?? 0
          const done = pickedQty >= li.quantity
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setPickLine({ lineItem: li, pickedSoFar: pickedQty })}
              className="w-full flex items-center gap-3 rounded-xl bg-[#1F2937] border border-[#2D3748] px-3.5 py-3 text-left hover:border-blue-500 transition-colors"
            >
              <LineItemThumb token={token} name={li.name} productId={li.product_id} />
              {done && <Check size={16} className="text-green-400 shrink-0" />}
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate">{li.name}</p>
                {li.sku_code && <p className="text-xs text-gray-500 truncate">{li.sku_code}</p>}
                {!li.sku_id && <p className="text-xs text-amber-400">Not linked to a product yet</p>}
              </div>
              <p className={`text-sm font-medium shrink-0 ${done ? 'text-green-400' : 'text-gray-300'}`}>
                {pickedQty} / {li.quantity}
              </p>
            </button>
          )
        })}

        {addedItems.length > 0 && (
          <div className="pt-2 space-y-2">
            <p className="text-xs text-gray-500 px-1">Added during picking:</p>
            {addedItems.map(s => (
              <div key={s.id} className="w-full flex items-center gap-3 rounded-xl bg-[#1F2937]/60 border border-[#2D3748] px-3.5 py-3">
                {s.action_type === 'freebie' ? <Gift size={16} className="text-purple-400 shrink-0" /> : <PackagePlus size={16} className="text-blue-400 shrink-0" />}
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{s.reason || 'New item'}</p>
                  <p className="text-xs text-gray-500 capitalize">{s.action_type === 'freebie' ? 'Freebie' : 'New item'}</p>
                </div>
                <p className="text-sm font-medium text-gray-300 shrink-0">×{s.quantity}</p>
              </div>
            ))}
          </div>
        )}

        <button
          type="button"
          onClick={() => setAddItemOpen(true)}
          className="w-full flex items-center justify-center gap-1.5 rounded-xl border border-dashed border-[#2D3748] px-3.5 py-3 text-sm text-blue-400 hover:border-blue-500 transition-colors"
        >
          <Gift size={14} /> Add freebie or new item
        </button>
      </div>

      <form onSubmit={handleScan} className="border-t border-[#2D3748] p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
        {candidates.length > 0 && (
          <div className="space-y-1.5">
            <p className="text-xs text-gray-500">Multiple matches -- pick one:</p>
            {candidates.map(c => (
              <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => handleCandidateSelect(c)} />
            ))}
          </div>
        )}
        <div className="flex gap-2">
          <input
            ref={codeInputRef}
            value={code}
            onChange={e => setCode(e.target.value)}
            placeholder="Scan or type barcode…"
            disabled={scanning || !!scanErrorAlert}
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
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
        <button
          type="submit" disabled={scanning || !code.trim() || !!scanErrorAlert}
          className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50"
        >
          {scanning ? 'Looking up…' : 'Scan'}
        </button>
      </form>

      {pickLine && (
        <PickLineSheet
          token={token}
          lineItem={pickLine.lineItem}
          pickedSoFar={pickLine.pickedSoFar}
          scans={(detail?.scans ?? []).filter(s => s.line_item_id === pickLine.lineItem.id)}
          onDeleteScan={handleDeleteScan}
          onCancel={() => setPickLine(null)}
          onPick={(quantity, reason) => {
            setPickLine(null)
            stage({ line_item_id: pickLine.lineItem.id, sku_id: pickLine.lineItem.sku_id ?? undefined, action_type: 'pick', quantity, reason })
          }}
          onSubstitute={substituteSkuId => {
            setPickLine(null)
            stage({ line_item_id: pickLine.lineItem.id, sku_id: substituteSkuId, action_type: 'substitute' })
          }}
        />
      )}

      {addItemOpen && (
        <AddItemSheet
          token={token}
          onCancel={() => setAddItemOpen(false)}
          onAdd={(skuId, actionType, quantity, priceMinorUnits, reason) => {
            setAddItemOpen(false)
            stage({ sku_id: skuId, action_type: actionType, quantity, price_minor_units: priceMinorUnits, reason })
          }}
        />
      )}

      {scanErrorAlert && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-6">
          <div className="w-full max-w-xs rounded-2xl bg-[#111827] border border-red-500/40 p-5 text-center">
            <p className="text-sm font-semibold text-red-400 mb-4">{scanErrorAlert}</p>
            <button
              type="button" onClick={dismissScanError}
              className="w-full py-3 rounded-xl bg-red-600 text-white font-semibold hover:bg-red-700"
            >
              OK
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

// Tap a line item row (or scan/confirm its own barcode) to either confirm
// it was picked, or -- when it's out of stock -- swap it for a different
// product entirely. Substitute always uses the ORIGINAL line's own
// quantity (whole-line swap only, never partial -- see
// CompletePickingUseCase's doc for why).
type PickSheetMode = 'pick' | 'substitute'

function PickLineSheet({ token, lineItem, pickedSoFar, scans, onDeleteScan, onCancel, onPick, onSubstitute }: {
  token: string
  lineItem: PickingSessionLineItem
  pickedSoFar: number
  scans: PickScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onPick: (quantity: number, reason?: string) => void
  onSubstitute: (substituteSkuId: string) => void
}) {
  const [mode, setMode] = useState<PickSheetMode>('pick')
  const [qty, setQty] = useState(String(Math.max(lineItem.quantity - pickedSoFar, 0)))
  const [reason, setReason] = useState('')
  const [substituteQuery, setSubstituteQuery] = useState('')
  const [substituteCandidates, setSubstituteCandidates] = useState<SKUCandidate[]>([])
  const [searchingSubstitute, setSearchingSubstitute] = useState(false)
  const n = Number(qty) || 0
  const remaining = Math.max(lineItem.quantity - pickedSoFar, 0)

  async function handleSubstituteSearch(e: React.FormEvent) {
    e.preventDefault()
    if (!substituteQuery.trim()) return
    setSearchingSubstitute(true)
    try {
      setSubstituteCandidates(await searchSKUCandidates(token, substituteQuery.trim()))
    } finally {
      setSearchingSubstitute(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{lineItem.name}</h2>
            <p className="text-xs text-gray-500 mt-0.5">Picked {pickedSoFar} of {lineItem.quantity}</p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        {scans.length > 0 && (
          <div className="px-5 pt-3 space-y-1.5">
            <p className="text-xs text-gray-500">Already staged for this line:</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center gap-2 rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-200">
                    {s.action_type === 'substitute' ? `Substituted × ${s.quantity}` : `Picked × ${s.quantity}`}
                  </p>
                  {s.reason && <p className="text-xs text-gray-500 truncate">{s.reason}</p>}
                </div>
                <button
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title="Delete this staged scan"
                  className="shrink-0 p-1.5 text-gray-500 hover:text-red-400"
                >
                  <X size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setMode('pick')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-[11px] font-medium leading-tight ${mode === 'pick' ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
            >
              <Check size={15} /> Pick
            </button>
            <button
              type="button"
              onClick={() => setMode('substitute')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-[11px] font-medium leading-tight ${mode === 'substitute' ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
            >
              <Repeat size={15} /> Substitute
            </button>
          </div>

          {mode === 'pick' ? (
            <>
              <QuantityStepper id="pick-quantity" label="Quantity to pick" value={qty} onChange={setQty} min={0} max={remaining} />
              <div>
                <label htmlFor="pick-reason" className="block text-xs text-gray-400 mb-1.5">Note (optional)</label>
                <input
                  id="pick-reason"
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <button
                onClick={() => onPick(n, reason || undefined)}
                disabled={n <= 0}
                className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold disabled:opacity-50"
              >
                Confirm pick
              </button>
            </>
          ) : (
            <>
              <p className="text-xs text-gray-500">
                Out of stock? Scan or search the replacement product -- it ships instead of this line, at the same quantity ({lineItem.quantity}).
              </p>
              <form onSubmit={handleSubstituteSearch} className="flex gap-2">
                <div className="relative flex-1">
                  <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
                  <input
                    value={substituteQuery}
                    onChange={e => setSubstituteQuery(e.target.value)}
                    placeholder="Search or scan barcode…"
                    className="w-full pl-8 pr-3 py-2.5 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <button type="submit" disabled={searchingSubstitute || !substituteQuery.trim()} className="px-4 rounded-xl bg-[#1F2937] border border-[#2D3748] text-sm text-gray-300 disabled:opacity-50">
                  {searchingSubstitute ? '…' : 'Find'}
                </button>
              </form>
              <div className="space-y-1.5">
                {substituteCandidates.map(c => (
                  <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => onSubstitute(c.sku.id)} />
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

// Scan/search any catalog product not on the order, then decide whether
// it's a no-charge freebie or a real, priced addition. Both create a real
// SalesOrderLineItem and decrement real stock when the session completes
// (see CompletePickingUseCase's freebie/new_item replay) -- Freebie forces
// price to 0 server-side regardless, so the price field is hidden rather
// than shown-and-ignored.
function AddItemSheet({ token, onCancel, onAdd }: {
  token: string
  onCancel: () => void
  onAdd: (skuId: string, actionType: PickScanActionType, quantity: number, priceMinorUnits: number | undefined, reason: string | undefined) => void
}) {
  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])
  const [selected, setSelected] = useState<SKUCandidate | null>(null)
  const [isFreebie, setIsFreebie] = useState(true)
  const [qty, setQty] = useState('1')
  const [price, setPrice] = useState('')
  const [reason, setReason] = useState('')
  const n = Number(qty) || 0
  const priceMinorUnits = Math.round((Number(price) || 0) * 100)

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    if (!query.trim()) return
    setSearching(true)
    try {
      setCandidates(await searchSKUCandidates(token, query.trim()))
    } finally {
      setSearching(false)
    }
  }

  function handleConfirm() {
    if (!selected) return
    onAdd(
      selected.sku.id,
      isFreebie ? 'freebie' : 'new_item',
      n,
      isFreebie ? undefined : priceMinorUnits,
      reason || selected.productName,
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">Add freebie or new item</h2>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {!selected ? (
            <>
              <form onSubmit={handleSearch} className="flex gap-2">
                <div className="relative flex-1">
                  <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
                  <input
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                    placeholder="Search or scan barcode…"
                    className="w-full pl-8 pr-3 py-2.5 text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                    autoFocus
                  />
                </div>
                <button type="submit" disabled={searching || !query.trim()} className="px-4 rounded-xl bg-[#1F2937] border border-[#2D3748] text-sm text-gray-300 disabled:opacity-50">
                  {searching ? '…' : 'Find'}
                </button>
              </form>
              <div className="space-y-1.5">
                {candidates.map(c => (
                  <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => { setSelected(c); setReason(c.productName) }} />
                ))}
              </div>
            </>
          ) : (
            <>
              <div className="flex items-center gap-3 rounded-xl bg-[#1F2937] border border-[#2D3748] px-3.5 py-3">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{selected.productName}</p>
                  <p className="text-xs text-gray-500 truncate">{selected.sku.sku_code}</p>
                </div>
                <button type="button" onClick={() => setSelected(null)} className="text-xs text-blue-400 shrink-0">Change</button>
              </div>

              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setIsFreebie(true)}
                  className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-[11px] font-medium leading-tight ${isFreebie ? 'bg-purple-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
                >
                  <Gift size={15} /> Freebie
                </button>
                <button
                  type="button"
                  onClick={() => setIsFreebie(false)}
                  className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-[11px] font-medium leading-tight ${!isFreebie ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
                >
                  <PackagePlus size={15} /> New item
                </button>
              </div>

              <QuantityStepper id="add-item-quantity" label="Quantity" value={qty} onChange={setQty} min={1} />

              {!isFreebie && (
                <div>
                  <label htmlFor="add-item-price" className="block text-xs text-gray-400 mb-1.5">Price (each)</label>
                  <input
                    id="add-item-price"
                    inputMode="decimal"
                    value={price}
                    onChange={e => setPrice(e.target.value)}
                    placeholder="0.00"
                    className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              )}

              <div>
                <label htmlFor="add-item-reason" className="block text-xs text-gray-400 mb-1.5">Name / note</label>
                <input
                  id="add-item-reason"
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <button
                onClick={handleConfirm}
                disabled={n <= 0 || (!isFreebie && priceMinorUnits < 0)}
                className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${isFreebie ? 'bg-purple-600' : 'bg-blue-600'}`}
              >
                Add to order
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
