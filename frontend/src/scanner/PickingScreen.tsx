import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, Camera, Gift, PackagePlus, Undo2, Lock } from 'lucide-react'
import { useSessionLock } from '../fulfillment/useSessionLock'
import { readClaims } from '../auth/jwtClaims'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { canTakeOverLocks, formatLockAge } from '../packing/lockDisplay'
import * as api from './api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import { playErrorSound } from './errorSound'
import { useWakeLock } from './useWakeLock'
import { computeMissingItems, pickedQuantityByLineItem } from './pickingMissingItems'
import type { PickingSessionDetail, PickingSessionLineItem, CompletePickingResult, StockShortfall, StockShortfallResolution } from './types'
import { StockShortfallDialog } from './StockShortfallDialog'
import type { SalesOrder } from '../oms/types'
import { Dialog } from '../components/common/Dialog'
import { PickLineSheet } from './PickingLineSheet'
import { ProductThumb, SCANNER_THUMB } from '../pms/ProductThumb'
import { addressesEqual, customerName, shippingAddressLine } from '../oms/salesOrderDisplay'
import { formatDate } from '../format/date'
import { isAuthError } from '../auth/isAuthError'
import { UNKNOWN_CUSTOMER_LABEL } from './scannerLabels'

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

  // Opening only views; the first scan takes the lock. While we hold it we
  // heartbeat it, and leaving the order releases it.
  const me = readClaims(token)
  const nameFor = useMemberNames(token)
  const { heldByOther } = useSessionLock({
    sessionId, me: me?.userId ?? '',
    lockedBy: detail?.session.locked_by, lockedAt: detail?.session.locked_at,
    active: !!detail && detail.session.status !== 'completed',
    touch: id => api.touchPickingSession(token, id),
    release: id => api.releasePickingSessionLock(token, id),
    onLost: () => setError('Another user has taken over this picking session'),
  })
  const readOnly = heldByOther !== null

  async function handleTakeOver() {
    try {
      await api.takeOverPickingSession(token, sessionId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to take over')
    }
  }
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
      await stage({ sku_id: skuId, action_type: 'pick', quantity })
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
          <button aria-label="Back to picking" onClick={() => setConfirmingFinish(false)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to picking">
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
            <span className="text-base text-gray-300">{customerName(order, UNKNOWN_CUSTOMER_LABEL)}</span>
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
            {startingPacking ? 'Starting…' : 'Continue to packing'}
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
        <button aria-label="Pause and go back" onClick={handlePause} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Picking</h1>
        <button
          onClick={() => setConfirmingFinish(true)} disabled={readOnly}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700 disabled:opacity-50"
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
          <span className="text-base text-gray-300">{customerName(order, UNKNOWN_CUSTOMER_LABEL)}</span>
        </div>
        {shippingAddressLine(order.shipping) && (
          <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
        {shippingAddressLine(order.billing) && !addressesEqual(order.billing, order.shipping) && (
          <p className="text-sm text-amber-400 mt-0.5">Billing: {shippingAddressLine(order.billing)}</p>
        )}
        <button
          type="button" onClick={() => setResetConfirmOpen(true)} disabled={readOnly}
          className="mt-0.5 -mb-2 py-2.5 flex items-center gap-1.5 text-sm text-red-400 disabled:opacity-50"
        >
          <Undo2 size={12} /> Start over
        </button>
      </div>

      {heldByOther && (
        <div role="status" className="flex items-center gap-2 px-4 py-2 border-b border-night-700 bg-amber-500/10 text-sm text-amber-200 shrink-0">
          <Lock size={14} className="shrink-0" />
          <span className="flex-1">
            {nameFor(heldByOther)} is picking this order{detail?.session.locked_since ? ` · ${formatLockAge(detail.session.locked_since)}` : ''} -- view only.
          </span>
          {canTakeOverLocks(me?.role) && (
            <button type="button" onClick={handleTakeOver} className="min-h-11 px-3 -my-2 rounded-lg bg-night-800 border border-night-700 text-gray-100 font-medium">
              Take over
            </button>
          )}
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
          const pickedQty = pickedByLineItem.get(li.id) ?? 0
          const done = pickedQty >= li.quantity
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setPickLine({ lineItem: li, pickedSoFar: pickedQty })}
              disabled={readOnly}
              className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-accent transition-colors"
            >
              <ProductThumb token={token} productId={li.product_id} appearance={SCANNER_THUMB} title={li.name} />
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
            disabled={scanning || readOnly || !!scanErrorAlert}
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus disabled:opacity-50"
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
        <button
          type="submit" disabled={scanning || !code.trim() || !!scanErrorAlert}
          className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
        >
          {scanning ? 'Looking up…' : 'Scan'}
        </button>
      </form>

      {pickLine && (
        <PickLineSheet
          lineItem={pickLine.lineItem}
          pickedSoFar={pickLine.pickedSoFar}
          scans={(detail?.scans ?? []).filter(s => s.line_item_id === pickLine.lineItem.id)}
          onDeleteScan={handleDeleteScan}
          onCancel={() => setPickLine(null)}
          onPick={(quantity, reason) => {
            setPickLine(null)
            stage({ sku_id: pickLine.lineItem.sku_id ?? undefined, action_type: 'pick', quantity, reason })
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
