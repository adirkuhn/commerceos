import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, Camera, X, Package, Repeat, Gift, PackagePlus, Search, Undo2 } from 'lucide-react'
import * as api from './api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { QuantityStepper } from './QuantityStepper'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import { playErrorSound } from './errorSound'
import { useWakeLock } from './useWakeLock'
import { computeMissingItems, pickedQuantityByLineItem } from './pickingMissingItems'
import type {
  PickingSessionDetail, PickingSessionLineItem, PickScan, CompletePickingResult,
  StockShortfall, StockShortfallResolution, PickScanActionType, SubstituteLine,
} from './types'
import { StockShortfallDialog } from './StockShortfallDialog'
import type { SalesOrder, Address } from '../oms/types'
import { Dialog } from '../components/common/Dialog'

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString()
}

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

// Same field set/reasoning as SalesOrders.tsx's own addressesEqual -- not
// phone, which legitimately differs (billing contact vs. delivery contact)
// without the addresses actually being different.
function addressesEqual(a?: Address, b?: Address): boolean {
  if (!a || !b) return false
  const fields: (keyof Address)[] = ['first_name', 'last_name', 'company', 'address_1', 'address_2', 'city', 'state', 'postcode', 'country']
  return fields.every(f => (a[f] ?? '') === (b[f] ?? ''))
}

// Falls back to a placeholder icon when the line isn't linked to a product
// yet, or the product has no media -- same convention as BookingScreen's
// own LineItemThumb for procurement bookings.
function LineItemThumb({ token, name, productId }: { token: string; name: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0 overflow-hidden" title={name}>
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
  // Hands off to the packing screen for this same order -- see Picking.tsx's
  // continueToPacking doc. Starts (or resumes) the packing session itself
  // (picking already has the order/token in hand) and lets ScannerApp just
  // switch screens with the result.
  onContinueToPacking: (sessionId: string, order: SalesOrder) => void
}

function isAuthError(err: unknown): boolean {
  return err instanceof Error && /401|unauthorized/i.test(err.message)
}

export function PickingScreen({ token, sessionId, order, onExit, onAuthError, onContinueToPacking }: PickingScreenProps) {
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
  const [stockShortfalls, setStockShortfalls] = useState<StockShortfall[] | null>(null)
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])
  const [pickLine, setPickLine] = useState<{ lineItem: PickingSessionLineItem; pickedSoFar: number } | null>(null)
  const [addItemOpen, setAddItemOpen] = useState(false)
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const [resetting, setResetting] = useState(false)
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

  // Heartbeat: refresh this session's lock every 60s while it stays open --
  // see Picking.tsx's identical effect for the full rationale.
  useEffect(() => {
    if (detail?.session.status === 'completed') return
    const interval = setInterval(() => {
      api.touchPickingSession(token, sessionId).catch(err => {
        if (isAuthError(err)) return onAuthError()
        if (err instanceof Error && /locked/i.test(err.message)) {
          setError('Another user has taken over this picking session')
        }
      })
    }, 60_000)
    return () => clearInterval(interval)
  }, [sessionId, detail?.session.status, token, onAuthError])
  useWakeLock()

  async function stage(input: Parameters<typeof api.scanPickItem>[2]): Promise<boolean> {
    setError('')
    setScanning(true)
    try {
      await api.scanPickItem(token, sessionId, input)
      await load()
      return true
    } catch (err) {
      if (isAuthError(err)) {
        onAuthError()
        return false
      }
      setError(err instanceof Error ? err.message : 'Scan failed')
      return false
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  async function stageSubstitute(input: Parameters<typeof api.substituteLineItem>[2]): Promise<boolean> {
    setError('')
    setScanning(true)
    try {
      await api.substituteLineItem(token, sessionId, input)
      await load()
      return true
    } catch (err) {
      if (isAuthError(err)) {
        onAuthError()
        return false
      }
      setError(err instanceof Error ? err.message : 'Substitute failed')
      return false
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  async function stageRefund(input: Parameters<typeof api.refundLineItem>[2]): Promise<boolean> {
    setError('')
    setScanning(true)
    try {
      await api.refundLineItem(token, sessionId, input)
      await load()
      return true
    } catch (err) {
      if (isAuthError(err)) {
        onAuthError()
        return false
      }
      setError(err instanceof Error ? err.message : 'Refund failed')
      return false
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
  // weightGrams is set only when the scanned code was a weight-embedded
  // (butcher/deli scale) barcode -- see ResolvePickScanCodeUseCase. The
  // pick quantity becomes the scanned package's actual weight (in kg)
  // instead of a flat 1 unit; a per_weight SKU can only ever reach this
  // function via that path (ScanPickItemUseCase rejects any other
  // resolution for one), so weightGrams is never nil here for one.
  async function routeResolvedSku(skuId: string, weightGrams?: number | null) {
    const quantity = weightGrams != null ? weightGrams / 1000 : 1
    const li = detail?.line_items.find(l => l.sku_id === skuId)
    if (li) {
      const pickedSoFar = pickedQuantityByLineItem(detail?.scans ?? []).get(li.id) ?? 0
      if (pickedSoFar >= li.quantity) {
        playErrorSound()
        setScanErrorAlert('Already fully picked')
        return
      }
      await stage({ line_item_id: li.id, sku_id: skuId, action_type: 'pick', quantity })
      return
    }
    // No line item already carries this sku_id -- still worth trying the
    // server: it may auto-link one of this order's still-unmatched Woo
    // lines (product name matches the scanned item) rather than this
    // always being a genuine "wrong order" scan (see
    // ScanPickItemUseCase.autoLinkUnmatchedLine).
    const ok = await stage({ sku_id: skuId, action_type: 'pick', quantity })
    if (!ok) {
      setError('')
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
      routeResolvedSku(resolved.sku_id, resolved.weight_grams)
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
    // A name/code text match carries no real weight -- same rule
    // ScanPickItemUseCase enforces server-side for a barcode/SKU-code
    // scan, applied here too so this fuzzy-search fallback can't bypass it.
    if (c.sku.pricing_mode === 'per_weight') {
      setCandidates([])
      playErrorSound()
      setScanErrorAlert('Priced by weight — scan its scale barcode, not a name/code match')
      return
    }
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

  async function handleResetSession() {
    setResetting(true)
    try {
      await api.resetPickingSession(token, sessionId)
      setResetConfirmOpen(false)
      setPickLine(null)
      setAddItemOpen(false)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to reset picking session')
    } finally {
      setResetting(false)
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

  const [startingPacking, setStartingPacking] = useState(false)

  async function handleContinueToPacking() {
    setStartingPacking(true)
    try {
      const session = await api.startOrResumePacking(token, order.id)
      onContinueToPacking(session.id, order)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to start packing')
      setStartingPacking(false)
    }
  }

  async function handleConfirmComplete(resolutions?: Record<string, StockShortfallResolution>) {
    setFinishing(true)
    try {
      const r = await api.completePicking(token, sessionId, resolutions)
      if (!r.completed) {
        setStockShortfalls(r.stock_shortfalls ?? [])
        return
      }
      setStockShortfalls(null)
      setResult(r)
      setConfirmingFinish(false)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to complete')
    } finally {
      setFinishing(false)
    }
  }

  if (stockShortfalls) {
    return (
      <StockShortfallDialog
        shortfalls={stockShortfalls}
        submitting={finishing}
        onCancel={() => setStockShortfalls(null)}
        onConfirm={resolutions => handleConfirmComplete(resolutions)}
      />
    )
  }

  if (confirmingFinish && detail) {
    const missing = computeMissingItems(detail.line_items, detail.scans)
    return (
      <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
          <button onClick={() => setConfirmingFinish(false)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to picking">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
        </div>
        {/* Customer/address stays visible here too, not just during active
            picking -- staff confirming completion should never have to go
            back just to double-check who this ships to (or create the label
            below without knowing the destination). */}
        <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
          <div className="flex items-baseline gap-1.5 flex-wrap">
            <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
            <span className="text-gray-400">·</span>
            <span className="text-base text-gray-300">{customerName(order)}</span>
          </div>
          {shippingAddressLine(order.shipping) && (
            <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
          )}
          {shippingAddressLine(order.billing) && !addressesEqual(order.billing, order.shipping) && (
            <p className="text-sm text-amber-400 mt-0.5">Billing: {shippingAddressLine(order.billing)}</p>
          )}
          {(order.shipping_method_title || order.shipping_method) && (
            <p className="text-sm text-gray-400 mt-0.5">Shipping: {order.shipping_method_title || order.shipping_method}</p>
          )}
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          {missing.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-8">Every line is fully picked.</p>
          ) : (
            <>
              <p className="text-sm text-amber-400">{missing.length} line(s) aren't fully picked yet:</p>
              {missing.map(m => (
                <div key={m.line_item_id} className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                  <p className="text-sm text-gray-200">{m.name}</p>
                  <p className="text-xs text-gray-400">expected {m.expected_qty} · picked {m.picked_qty}</p>
                </div>
              ))}
            </>
          )}
        </div>
        <div className="p-4 border-t border-night-700 shrink-0 space-y-2">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={() => handleConfirmComplete()}
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
          <h1 className="text-lg font-semibold">Order picked</h1>
          {!result.missing_items?.length && <p className="text-sm text-gray-400">Every line was fully picked.</p>}
        </div>
        {!!result.missing_items?.length && (
          <div className="flex-1 overflow-y-auto px-6 pb-4 space-y-2">
            <p className="text-sm text-amber-400">{result.missing_items.length} line(s) weren't fully picked:</p>
            {result.missing_items.map(m => (
              <div key={m.line_item_id} className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <p className="text-sm text-gray-200">{m.name}</p>
                <p className="text-xs text-gray-400">expected {m.expected_qty} · picked {m.picked_qty}</p>
              </div>
            ))}
          </div>
        )}
        <div className="p-6 shrink-0 space-y-2">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={handleContinueToPacking} disabled={startingPacking}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {startingPacking ? 'Starting…' : 'Continue to Packing'}
          </button>
          <button onClick={onExit} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
            Done
          </button>
        </div>
      </div>
    )
  }

  const pickedByLineItem = pickedQuantityByLineItem(detail?.scans ?? [])
  const addedItems = (detail?.scans ?? []).filter(s => s.action_type === 'freebie' || s.action_type === 'new_item')

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button onClick={handlePause} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Picking</h1>
        <button
          onClick={() => setConfirmingFinish(true)}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700"
        >
          Finish
        </button>
      </div>

      {/* Order info -- its own full-width band, separate from the control
          row above, so the address has room to wrap instead of getting
          truncated next to the back/Finish buttons. */}
      <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
        <div className="flex items-baseline gap-1.5 flex-wrap">
          <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
          <span className="text-gray-400">·</span>
          <span className="text-base text-gray-300">{customerName(order)}</span>
        </div>
        {shippingAddressLine(order.shipping) && (
          <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
        {shippingAddressLine(order.billing) && !addressesEqual(order.billing, order.shipping) && (
          <p className="text-sm text-amber-400 mt-0.5">Billing: {shippingAddressLine(order.billing)}</p>
        )}
        <button
          type="button" onClick={() => setResetConfirmOpen(true)}
          className="mt-0.5 -mb-2 py-2.5 flex items-center gap-1.5 text-sm text-red-400"
        >
          <Undo2 size={12} /> Start over
        </button>
      </div>

      {cameraOpen && (
        <div className="px-4 pt-3 shrink-0">
          <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} frameShape="rect" variant="inline" />
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-400 py-8">Loading…</p>}
        {detail?.line_items.map(li => {
          const pickedQty = pickedByLineItem.get(li.id) ?? 0
          const done = pickedQty >= li.quantity
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setPickLine({ lineItem: li, pickedSoFar: pickedQty })}
              className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-accent transition-colors"
            >
              <LineItemThumb token={token} name={li.name} productId={li.product_id} />
              {done && <Check size={16} className="text-green-400 shrink-0" />}
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate">{li.name}</p>
                {li.sku_code && <p className="text-xs text-gray-400 truncate">{li.sku_code}</p>}
                {li.sold_expiry_date && (
                  <p className="text-xs text-gray-400 truncate" title="Reported by WooCommerce at checkout -- may not match the batch actually allocated">
                    Sold exp. {formatDate(li.sold_expiry_date)}
                  </p>
                )}
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
            <p className="text-xs text-gray-400 px-1">Added during picking:</p>
            {addedItems.map(s => (
              <div key={s.id} className="w-full flex items-center gap-3 rounded-xl bg-night-800/60 border border-night-700 px-3.5 py-3">
                {s.action_type === 'freebie' ? <Gift size={16} className="text-purple-400 shrink-0" /> : <PackagePlus size={16} className="text-accent shrink-0" />}
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{s.reason || 'New item'}</p>
                  <p className="text-xs text-gray-400 capitalize">{s.action_type === 'freebie' ? 'Freebie' : 'New item'}</p>
                </div>
                <p className="text-sm font-medium text-gray-300 shrink-0">×{s.quantity}</p>
              </div>
            ))}
          </div>
        )}

        <button
          type="button"
          onClick={() => setAddItemOpen(true)}
          className="w-full flex items-center justify-center gap-1.5 rounded-xl border border-dashed border-night-700 px-3.5 py-3 text-sm text-accent hover:border-accent transition-colors"
        >
          <Gift size={14} /> Add freebie or new item
        </button>
      </div>

      <form onSubmit={handleScan} className="border-t border-night-700 p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
        {candidates.length > 0 && (
          <div className="space-y-1.5">
            <p className="text-xs text-gray-400">Multiple matches -- pick one:</p>
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
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus disabled:opacity-50"
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
        <button
          type="submit" disabled={scanning || !code.trim() || !!scanErrorAlert}
          className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
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
          onSubstitute={lines => {
            setPickLine(null)
            stageSubstitute({ line_item_id: pickLine.lineItem.id, lines })
          }}
          onRefund={(quantity, reason) => {
            setPickLine(null)
            stageRefund({ line_item_id: pickLine.lineItem.id, quantity, reason })
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
        <Dialog onClose={dismissScanError} label="Scan error" initialFocus="dialog" className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-6">
          <div className="w-full max-w-xs rounded-2xl bg-night-900 border border-red-500/40 p-5 text-center">
            <p className="text-sm font-semibold text-red-400 mb-4">{scanErrorAlert}</p>
            <button
              type="button" onClick={dismissScanError}
              className="w-full py-3 rounded-xl bg-red-600 text-white font-semibold hover:bg-red-700"
            >
              OK
            </button>
          </div>
        </Dialog>
      )}

      {resetConfirmOpen && (
        <Dialog onClose={() => setResetConfirmOpen(false)} label="Start over?" dismissible={!resetting} className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-6">
          <div className="w-full max-w-xs rounded-2xl bg-night-900 border border-night-700 p-5 text-center">
            <p className="text-sm font-semibold mb-1">Start over?</p>
            <p className="text-xs text-gray-400 mb-4">
              This discards every item picked so far on this order and starts the session fresh. This can't be undone.
            </p>
            <div className="flex gap-2">
              <button
                type="button" onClick={() => setResetConfirmOpen(false)} disabled={resetting}
                className="flex-1 py-3 rounded-xl border border-night-700 text-sm font-medium text-gray-300 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button" onClick={handleResetSession} disabled={resetting}
                className="flex-1 py-3 rounded-xl bg-red-600 text-white text-sm font-semibold hover:bg-red-700 disabled:opacity-50"
              >
                {resetting ? 'Resetting…' : 'Start over'}
              </button>
            </div>
          </div>
        </Dialog>
      )}
    </div>
  )
}

// Tap a line item row (or scan/confirm its own barcode) to either confirm
// it was picked, or -- when it's out of stock -- swap it for one or more
// replacement products. A single original unit can be split across
// several substitute SKUs (e.g. 1 unit for 3 of one cheaper item + 4 of
// another) -- each row picks its own SKU, how many of it is being added,
// how much of the original line it covers, and whether that covered share
// goes back to stock. The rows' covered quantities must sum to the
// original line's own quantity exactly before submitting.
type PickSheetMode = 'pick' | 'substitute' | 'refund'

let substituteRowKeySeq = 0
function nextSubstituteRowKey(): string { return `sub-row-${++substituteRowKeySeq}` }

interface SubstituteRowState {
  key: string
  query: string
  candidates: SKUCandidate[]
  searching: boolean
  selected?: SKUCandidate
  substituteQty: string
  coveredQty: string
  returnToStock: boolean
}

function newSubstituteRow(): SubstituteRowState {
  return { key: nextSubstituteRowKey(), query: '', candidates: [], searching: false, substituteQty: '1', coveredQty: '', returnToStock: false }
}

function PickLineSheet({ token, lineItem, pickedSoFar, scans, onDeleteScan, onCancel, onPick, onSubstitute, onRefund }: {
  token: string
  lineItem: PickingSessionLineItem
  pickedSoFar: number
  scans: PickScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onPick: (quantity: number, reason?: string) => void
  onSubstitute: (lines: SubstituteLine[]) => void
  onRefund: (quantity: number, reason?: string) => void
}) {
  const [mode, setMode] = useState<PickSheetMode>('pick')
  const [qty, setQty] = useState(String(Math.max(lineItem.quantity - pickedSoFar, 0)))
  const [reason, setReason] = useState('')
  const [substituteRows, setSubstituteRows] = useState<SubstituteRowState[]>([newSubstituteRow()])
  const [refundQty, setRefundQty] = useState(String(Math.max(lineItem.quantity - pickedSoFar, 0)))
  const [refundReason, setRefundReason] = useState('')
  const n = Number(qty) || 0
  const refundN = Number(refundQty) || 0
  const remaining = Math.max(lineItem.quantity - pickedSoFar, 0)

  const coveredTotal = substituteRows.reduce((sum, r) => sum + (Number(r.coveredQty) || 0), 0)
  // Must cover exactly what's left, not the line's full original quantity
  // -- a line already partially picked can still have the rest substituted.
  const substituteReady = substituteRows.length > 0
    && substituteRows.every(r => r.selected && Number(r.substituteQty) > 0 && Number(r.coveredQty) > 0)
    && coveredTotal === remaining

  function updateRow(key: string, patch: Partial<SubstituteRowState>) {
    setSubstituteRows(rows => rows.map(r => (r.key === key ? { ...r, ...patch } : r)))
  }

  async function handleRowSearch(key: string, e: React.FormEvent) {
    e.preventDefault()
    const row = substituteRows.find(r => r.key === key)
    if (!row || !row.query.trim()) return
    updateRow(key, { searching: true })
    try {
      const candidates = await searchSKUCandidates(token, row.query.trim())
      updateRow(key, { candidates, searching: false })
    } catch {
      updateRow(key, { searching: false })
    }
  }

  function handleSubmitSubstitute() {
    if (!substituteReady) return
    onSubstitute(substituteRows.map(r => ({
      sku_id: r.selected!.sku.id,
      substitute_quantity: Number(r.substituteQty),
      original_quantity_covered: Number(r.coveredQty),
      return_to_stock: r.returnToStock,
    })))
  }

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{lineItem.name}</h2>
            <p className="text-xs text-gray-400 mt-0.5">Picked {pickedSoFar} of {lineItem.quantity}</p>
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
                  <p className="text-sm text-gray-200">
                    {s.action_type === 'substitute'
                      ? `Substituted × ${s.quantity} (covers ${s.original_quantity_covered ?? '?'})`
                      : s.action_type === 'refund'
                        ? `Refunded × ${s.quantity}`
                        : `Picked × ${s.quantity}`}
                  </p>
                  {s.reason && <p className="text-xs text-gray-400 truncate">{s.reason}</p>}
                </div>
                <button
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title="Delete this staged scan"
                  className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                >
                  <X size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            <button aria-pressed={mode === 'pick'}
              type="button"
              onClick={() => setMode('pick')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${mode === 'pick' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
            >
              <Check size={15} /> Pick
            </button>
            <button aria-pressed={mode === 'substitute'}
              type="button"
              onClick={() => setMode('substitute')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${mode === 'substitute' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
            >
              <Repeat size={15} /> Substitute
            </button>
            <button aria-pressed={mode === 'refund'}
              type="button"
              onClick={() => setMode('refund')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${mode === 'refund' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
            >
              <Undo2 size={15} /> Refund
            </button>
          </div>

          {mode === 'refund' ? (
            <>
              <p className="text-xs text-gray-400">
                Can't fulfill this and don't want to substitute? Refund part or all of this line's quantity
                ({lineItem.quantity}) instead -- money only moves once the session is completed.
              </p>
              <QuantityStepper id="refund-quantity" label="Quantity to refund" value={refundQty} onChange={setRefundQty} min={0} max={lineItem.quantity} />
              <div>
                <label htmlFor="refund-reason" className="block text-xs text-gray-400 mb-1.5">Reason (optional)</label>
                <input
                  id="refund-reason"
                  value={refundReason}
                  onChange={e => setRefundReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                />
              </div>
              <button
                onClick={() => onRefund(refundN, refundReason || undefined)}
                disabled={refundN <= 0}
                className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
              >
                Confirm refund
              </button>
            </>
          ) : mode === 'pick' ? (
            <>
              <QuantityStepper id="pick-quantity" label="Quantity to pick" value={qty} onChange={setQty} min={0} max={remaining} />
              <div>
                <label htmlFor="pick-reason" className="block text-xs text-gray-400 mb-1.5">Note (optional)</label>
                <input
                  id="pick-reason"
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                />
              </div>
              <button
                onClick={() => onPick(n, reason || undefined)}
                disabled={n <= 0}
                className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
              >
                Confirm pick
              </button>
            </>
          ) : (
            <>
              <p className="text-xs text-gray-400">
                Out of stock? Replace it with one or more products -- each row's "covers" quantity must add up to
                this line's remaining quantity ({remaining}) before you can confirm.
              </p>
              {substituteRows.map((row, i) => (
                <div key={row.key} className="rounded-xl bg-night-800 border border-night-700 p-3 space-y-2">
                  {row.selected ? (
                    <div className="flex items-center gap-2">
                      <div className="flex-1 min-w-0 text-sm text-gray-200 truncate">{row.selected.sku.sku_code}</div>
                      <button
                        type="button"
                        onClick={() => updateRow(row.key, { selected: undefined, candidates: [] })}
                        className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                        title="Change product"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  ) : (
                    <>
                      <form onSubmit={e => handleRowSearch(row.key, e)} className="flex gap-2">
                        <div className="relative flex-1">
                          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                          <input
                            value={row.query}
                            onChange={e => updateRow(row.key, { query: e.target.value })}
                            placeholder="Search or scan barcode…"
                            className="w-full pl-8 pr-3 py-2 text-sm rounded-xl bg-night-900 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                          />
                        </div>
                        <button type="submit" disabled={row.searching || !row.query.trim()} className="px-4 rounded-xl bg-night-900 border border-night-700 text-sm text-gray-300 disabled:opacity-50">
                          {row.searching ? '…' : 'Find'}
                        </button>
                      </form>
                      <div className="space-y-1.5">
                        {row.candidates.map(c => (
                          <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => updateRow(row.key, { selected: c, candidates: [] })} />
                        ))}
                      </div>
                    </>
                  )}
                  {row.selected && (
                    <>
                      <div className="grid grid-cols-2 gap-2">
                        <label className="block">
                          <span className="block text-2xs text-gray-400 mb-1">Qty added</span>
                          <input
                            type="number" min={1} value={row.substituteQty}
                            onChange={e => updateRow(row.key, { substituteQty: e.target.value })}
                            className="w-full px-3 py-2 text-sm rounded-xl bg-night-900 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                          />
                        </label>
                        <label className="block">
                          <span className="block text-2xs text-gray-400 mb-1">Covers (of original)</span>
                          <input
                            type="number" min={1} value={row.coveredQty}
                            onChange={e => updateRow(row.key, { coveredQty: e.target.value })}
                            className="w-full px-3 py-2 text-sm rounded-xl bg-night-900 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                          />
                        </label>
                      </div>
                      <label className="flex items-center gap-2 text-xs text-gray-400">
                        <input
                          type="checkbox" checked={row.returnToStock}
                          onChange={e => updateRow(row.key, { returnToStock: e.target.checked })}
                          className="rounded border-night-700 bg-night-900"
                        />
                        Return original quantity to stock
                      </label>
                    </>
                  )}
                  {substituteRows.length > 1 && (
                    <button
                      type="button"
                      onClick={() => setSubstituteRows(rows => rows.filter((_, idx) => idx !== i))}
                      className="text-xs text-gray-400 hover:text-red-400"
                    >
                      Remove this row
                    </button>
                  )}
                </div>
              ))}
              <button
                type="button"
                onClick={() => setSubstituteRows(rows => [...rows, newSubstituteRow()])}
                className="w-full py-2 rounded-xl border border-dashed border-night-700 text-sm text-gray-400 hover:text-gray-200"
              >
                + Add another substitute
              </button>
              <p className="text-xs text-gray-400 text-center">
                Covers {coveredTotal} of {remaining}
              </p>
              <button
                onClick={handleSubmitSubstitute}
                disabled={!substituteReady}
                className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
              >
                Confirm substitute
              </button>
            </>
          )}
        </div>
      </div>
    </Dialog>
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
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">Add freebie or new item</h2>
          <button onClick={onCancel} title="Close" className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {!selected ? (
            <>
              <form onSubmit={handleSearch} className="flex gap-2">
                <div className="relative flex-1">
                  <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                    placeholder="Search or scan barcode…"
                    className="w-full pl-8 pr-3 py-2.5 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                    autoFocus
                  />
                </div>
                <button type="submit" disabled={searching || !query.trim()} className="px-4 rounded-xl bg-night-800 border border-night-700 text-sm text-gray-300 disabled:opacity-50">
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
              <div className="flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{selected.productName}</p>
                  <p className="text-xs text-gray-400 truncate">{selected.sku.sku_code}</p>
                </div>
                <button type="button" onClick={() => setSelected(null)} className="text-xs text-accent shrink-0">Change</button>
              </div>

              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setIsFreebie(true)}
                  className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${isFreebie ? 'bg-purple-600 text-white' : 'bg-night-800 text-gray-400'}`}
                >
                  <Gift size={15} /> Freebie
                </button>
                <button
                  type="button"
                  onClick={() => setIsFreebie(false)}
                  className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${!isFreebie ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
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
                    className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                  />
                </div>
              )}

              <div>
                <label htmlFor="add-item-reason" className="block text-xs text-gray-400 mb-1.5">Name / note</label>
                <input
                  id="add-item-reason"
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                />
              </div>

              <button
                onClick={handleConfirm}
                disabled={n <= 0 || (!isFreebie && priceMinorUnits < 0)}
                className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${isFreebie ? 'bg-purple-600' : 'bg-accent-solid'}`}
              >
                Add to order
              </button>
            </>
          )}
        </div>
      </div>
    </Dialog>
  )
}
