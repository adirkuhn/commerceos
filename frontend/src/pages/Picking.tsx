import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { ScanBarcode, CheckCircle, Package, Search, PauseCircle, Repeat, Gift, PackagePlus, X, Trash2, ArrowLeft, Undo2, PackageCheck } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as omsApi from '../oms/api'
import type { SalesOrder } from '../oms/types'
import * as pickingApi from '../scanner/api'
import type { PickingSessionDetail, PickingSessionLineItem, PickScan, CompletePickingResult, StockShortfall, StockShortfallResolution } from '../scanner/types'
import { pickedQuantityByLineItem, computeMissingItems } from '../scanner/pickingMissingItems'
import { StockShortfallDialog } from '../scanner/StockShortfallDialog'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { playErrorSound } from '../scanner/errorSound'
import type { Address } from '../oms/types'
import { useRoutedSession } from '../oms/useRoutedSession'
import { SessionErrorPanel } from '../oms/SessionErrorPanel'
import { Dialog } from '../components/common/Dialog'

// Falls back to a placeholder icon when the line isn't linked to a product
// yet, or the product has no media -- same convention as the scanner app's
// own PickingScreen/BookingScreen thumbnails.
function LineItemThumb({ token, productId }: { token: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-10 h-10 rounded-lg bg-hover flex items-center justify-center shrink-0 overflow-hidden">
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={18} className="text-ink-4" />
      )}
    </div>
  )
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

function shippingAddressLine(address?: Address): string {
  if (!address) return ''
  const streetAndLocality = [address.address_1, address.address_2, [address.postcode, address.city].filter(Boolean).join(' ')]
    .filter(Boolean).join(', ')
  return [streetAndLocality, address.country].filter(Boolean).join(', ')
}


const ACTION_LABEL: Record<PickScan['action_type'], string> = {
  pick: 'Picked', substitute: 'Substituted', freebie: 'Freebie', new_item: 'New item', refund: 'Refunded',
}

// One row of a substitute batch -- each row scans/types its own
// replacement code, how many of it is being added, how much of the
// original line it covers, and whether that covered share goes back to
// stock. Rows' coveredQty must sum to the original line's own quantity
// exactly before submitting (see SubstituteLineItemUseCase).
interface SubstituteRowState {
  key: number
  code: string
  substituteQty: string
  coveredQty: string
  returnToStock: boolean
}

let substituteRowKeySeq = 0
function newSubstituteRow(): SubstituteRowState {
  return { key: ++substituteRowKeySeq, code: '', substituteQty: '1', coveredQty: '', returnToStock: false }
}

const PICKING_PATH = '/picking'

// Each open session has its own URL (/picking/:sessionId) so a refresh or the
// browser Back button keeps the picker on the order they were working on.
// Session id rather than order id: reopening is a read (get session -> get
// its order), never a startOrResume, which would fail once the order is
// fulfilled by completing the session.
export default function Picking() {
  const { token } = useAuth()

  const [orders, setOrders] = useState<SalesOrder[]>([])
  const [ordersLoading, setOrdersLoading] = useState(true)
  const [ordersError, setOrdersError] = useState('')
  const [search, setSearch] = useState('')

  const [activeOrder, setActiveOrder] = useState<SalesOrder | null>(null)
  const [session, setSession] = useState<PickingSessionDetail | null>(null)
  const [sessionLoading, setSessionLoading] = useState(false)
  const [error, setError] = useState('')

  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)
  const codeRef = useRef<HTMLInputElement>(null)

  const [pickLine, setPickLine] = useState<{ lineItem: PickingSessionLineItem; pickedSoFar: number } | null>(null)
  const [pickQty, setPickQty] = useState(1)
  const [pickPanelMode, setPickPanelMode] = useState<'pick' | 'substitute' | 'refund'>('pick')
  const [substituteRows, setSubstituteRows] = useState<SubstituteRowState[]>([])
  const [refundQty, setRefundQty] = useState('1')
  const [refundReason, setRefundReason] = useState('')

  const [addItemOpen, setAddItemOpen] = useState<{ skuId: string; productName: string } | null>(null)
  const [addItemQty, setAddItemQty] = useState(1)
  const [addItemPrice, setAddItemPrice] = useState('0')
  const [addItemFreebie, setAddItemFreebie] = useState(true)

  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const [resetting, setResetting] = useState(false)

  const [pausing, setPausing] = useState(false)
  const [completing, setCompleting] = useState(false)
  const [completeResult, setCompleteResult] = useState<CompletePickingResult | null>(null)
  // Complete no longer fires the API directly -- it opens this review step
  // first (divergences + customer/address, always shown even with none) so
  // gaps can't silently slip through into a "fulfilled" order. Only
  // handleConfirmComplete (below) actually calls completePicking.
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  // Set when completePicking comes back with completed:false -- a
  // substitute/freebie/new-item pick's SKU is short on tracked stock, see
  // StockShortfallDialog. Resubmitting with staff's per-item resolutions
  // either clears this (completed) or replaces it (still short -- shows
  // the same dialog again with fresh numbers).
  const [stockShortfalls, setStockShortfalls] = useState<StockShortfall[] | null>(null)

  const navigate = useNavigate()

  const loadOrders = useCallback(async () => {
    if (!token) return
    setOrdersLoading(true)
    setOrdersError('')
    try {
      const result = await omsApi.listSalesOrders(token, { fulfilled: false, status: ['pending', 'processing', 'on-hold'], perPage: 100 })
      setOrders(result.salesOrders)
    } catch (err) {
      setOrdersError(err instanceof Error ? err.message : 'Failed to load orders')
    } finally {
      setOrdersLoading(false)
    }
  }, [token])

  useEffect(() => { loadOrders() }, [loadOrders])

  const loadSession = useCallback(async (sessionId: string) => {
    if (!token) return
    try {
      setSession(await pickingApi.getPickingSession(token, sessionId))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load picking session')
    } finally {
      setSessionLoading(false)
    }
  }, [token])

  useEffect(() => { codeRef.current?.focus() }, [session, pickLine, addItemOpen])

  // Heartbeat: refresh this session's lock every 60s while it's open, so a
  // long pick doesn't outlive the backend's PickingLockTTL and get taken
  // over by someone else mid-pack. A 409 means it already was -- surface it
  // once and stop touching (further scans/completes will 409 too and
  // report the same thing).
  useEffect(() => {
    const sessionId = session?.session.id
    if (!sessionId || !token || session?.session.status === 'completed') return
    const interval = setInterval(() => {
      pickingApi.touchPickingSession(token, sessionId).catch(err => {
        if (err instanceof Error && /locked/i.test(err.message)) {
          setError('Another user has taken over this picking session')
        }
      })
    }, 60_000)
    return () => clearInterval(interval)
  }, [session?.session.id, session?.session.status, token])

  function clearSelection() {
    setActiveOrder(null)
    setSession(null)
    setSessionLoading(false)
    setCompleteResult(null)
    setStockShortfalls(null)
    setError('')
    setConfirmingFinish(false)
    setPickLine(null)
    setAddItemOpen(null)
  }

  const { urlSessionId, showSession, backToList: backToOrders } = useRoutedSession({
    basePath: PICKING_PATH,
    token,
    loadDetail: pickingApi.getPickingSession,
    currentOrder: activeOrder,
    onLoading: () => { setSessionLoading(true); setError('') },
    onLoaded: (order, detail) => { setActiveOrder(order); setSession(detail); setSessionLoading(false) },
    onError: message => { setError(message); setSessionLoading(false) },
    onCleared: clearSelection,
  })

  async function selectOrder(order: SalesOrder) {
    setError('')
    setCompleteResult(null)
    setStockShortfalls(null)
    setPickLine(null)
    setAddItemOpen(null)
    // Switching to a different order while one is actively in progress is
    // just pause-then-resume -- same mechanic as the scanner app.
    if (session && session.session.status === 'in_progress' && session.session.sales_order_id !== order.id && token) {
      try { await pickingApi.pausePicking(token, session.session.id) } catch { /* best-effort */ }
    }
    setActiveOrder(order)
    setSession(null)
    setSessionLoading(true)
    try {
      const newSession = await pickingApi.startOrResumePicking(token ?? '', order.id)
      if (showSession(newSession.id)) await loadSession(newSession.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start picking')
      setSessionLoading(false)
    }
  }

  // Hands off to the packing screen for this same order -- picking's job is
  // done (see CompletePickingUseCase), packing (box + label) happens there
  // now, whether that's the same person continuing immediately or a
  // different person picking it up later from the packing screen's own
  // order list.
  function continueToPacking() {
    if (!activeOrder) return
    navigate(`/packing?order=${activeOrder.id}`)
  }

  // A scanned/typed code that matches one of this order's own line items
  // marks one unit picked immediately -- no confirm sheet, just an
  // increment (see the picking screens' shared design doc for why: the
  // sheet is now reserved for a deliberate tap on the line itself, to pick
  // a specific quantity or substitute). Already-fully-picked (over-scan) or
  // resolved-but-not-on-this-order both block further scanning behind
  // scanErrorAlert until the picker explicitly confirms the error --
  // mis-scans must never silently fall through to the freebie/new-item
  // flow.
  async function handleScanSubmit() {
    const scannedCode = code.trim()
    if (!scannedCode || !session || !token || scanErrorAlert) return
    setError('')
    setBusy(true)
    try {
      const resolved = await pickingApi.resolvePickScanCode(token, session.session.id, scannedCode)
      setCode('')
      const li = session.line_items.find(l => l.sku_id === resolved.sku_id)
      if (li) {
        const pickedSoFar = pickedQuantityByLineItem(session.scans).get(li.id) ?? 0
        if (pickedSoFar >= li.quantity) {
          playErrorSound()
          setScanErrorAlert('Already fully picked')
          return
        }
      }
      // No line item already carries this sku_id -- still worth trying the
      // server: it may auto-link one of this order's still-unmatched Woo
      // lines (product name matches the scanned item) rather than this
      // always being a genuine "wrong order" scan (see
      // ScanPickItemUseCase.autoLinkUnmatchedLine).
      try {
        await pickingApi.scanPickItem(token, session.session.id, {
          sku_id: resolved.sku_id, line_item_id: li?.id, action_type: 'pick', quantity: 1,
        })
        await loadSession(session.session.id)
      } catch {
        playErrorSound()
        setScanErrorAlert('Item not on this order')
      }
    } catch {
      // Code didn't resolve to any known SKU at all -- must block the same
      // way as the "item not on this order" / "already fully picked" cases
      // above, not just show inline text a picker can scan straight past
      // (see PickingScreen.tsx's resolveAndRoute, which already blocks here).
      playErrorSound()
      setScanErrorAlert('Barcode not recognized')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeRef.current?.focus()
  }

  // Separate from handleScanSubmit -- this fires from inside the "Add item"
  // panel's own scan box, opened explicitly via the toolbar button. Unlike
  // the main scan bar, it must never divert to the Pick sheet just because
  // the resolved SKU happens to already be on the order (the picker chose
  // "Add item" on purpose, e.g. to add a second unit as a freebie).
  async function handleAddItemScanSubmit() {
    const scannedCode = code.trim()
    if (!scannedCode || !session || !token) return
    setError('')
    setBusy(true)
    try {
      const resolved = await pickingApi.resolvePickScanCode(token, session.session.id, scannedCode)
      setCode('')
      setAddItemOpen({ skuId: resolved.sku_id, productName: resolved.product_name })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Scan not recognized')
    } finally {
      setBusy(false)
    }
  }

  function openLineForPick(li: PickingSessionLineItem, pickedSoFar: number) {
    if (!li.sku_id) return // an unmatched Woo-import line has no SKU to pick against -- must be resolved elsewhere first
    setPickLine({ lineItem: li, pickedSoFar })
    setPickQty(Math.max(1, li.quantity - pickedSoFar))
    setPickPanelMode('pick')
    setSubstituteRows([newSubstituteRow()])
    setRefundQty(String(Math.max(1, li.quantity - pickedSoFar)))
    setRefundReason('')
  }

  async function confirmPick() {
    if (!pickLine || !session || !token || !pickLine.lineItem.sku_id) return
    setBusy(true)
    setError('')
    try {
      await pickingApi.scanPickItem(token, session.session.id, {
        sku_id: pickLine.lineItem.sku_id, line_item_id: pickLine.lineItem.id, action_type: 'pick', quantity: pickQty,
      })
      setPickLine(null)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record pick')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  function updateSubstituteRow(key: number, patch: Partial<SubstituteRowState>) {
    setSubstituteRows(rows => rows.map(r => (r.key === key ? { ...r, ...patch } : r)))
  }

  const substituteCoveredTotal = substituteRows.reduce((sum, r) => sum + (Number(r.coveredQty) || 0), 0)
  // Must cover exactly what's left, not the line's full original quantity
  // -- a line already partially picked can still have the rest substituted.
  const substituteRemaining = pickLine ? pickLine.lineItem.quantity - pickLine.pickedSoFar : 0
  const substituteReady = !!pickLine && substituteRows.length > 0
    && substituteRows.every(r => r.code.trim() && Number(r.substituteQty) > 0 && Number(r.coveredQty) > 0)
    && substituteCoveredTotal === substituteRemaining

  async function confirmSubstitute() {
    if (!pickLine || !session || !token || !substituteReady) return
    setBusy(true)
    setError('')
    try {
      const lines = await Promise.all(substituteRows.map(async r => {
        const resolved = await pickingApi.resolvePickScanCode(token, session.session.id, r.code.trim())
        return {
          sku_id: resolved.sku_id, substitute_quantity: Number(r.substituteQty),
          original_quantity_covered: Number(r.coveredQty), return_to_stock: r.returnToStock,
        }
      }))
      await pickingApi.substituteLineItem(token, session.session.id, { line_item_id: pickLine.lineItem.id, lines })
      setPickLine(null)
      setPickPanelMode('pick')
      setSubstituteRows([])
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record substitute')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  const refundQtyNum = Number(refundQty) || 0
  const refundReady = !!pickLine && refundQtyNum > 0 && refundQtyNum <= pickLine.lineItem.quantity

  async function confirmRefund() {
    if (!pickLine || !session || !token || !refundReady) return
    setBusy(true)
    setError('')
    try {
      await pickingApi.refundLineItem(token, session.session.id, {
        line_item_id: pickLine.lineItem.id, quantity: refundQtyNum, reason: refundReason || undefined,
      })
      setPickLine(null)
      setPickPanelMode('pick')
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record refund')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  async function confirmAddItem() {
    if (!addItemOpen || !session || !token) return
    setBusy(true)
    setError('')
    try {
      const priceMinorUnits = Math.round(parseFloat(addItemPrice || '0') * 100)
      await pickingApi.scanPickItem(token, session.session.id, {
        sku_id: addItemOpen.skuId,
        action_type: addItemFreebie ? 'freebie' : 'new_item',
        quantity: addItemQty,
        price_minor_units: addItemFreebie ? undefined : priceMinorUnits,
        reason: addItemOpen.productName,
      })
      setAddItemOpen(null)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add item')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  async function handleDeleteScan(scanId: string) {
    if (!session || !token) return
    setError('')
    try {
      await pickingApi.deletePickScan(token, session.session.id, scanId)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove scan')
    }
  }

  async function handlePause() {
    if (!session || !token) return
    setPausing(true)
    setError('')
    try {
      await pickingApi.pausePicking(token, session.session.id)
      backToOrders()
      await loadOrders()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to pause')
    } finally {
      setPausing(false)
    }
  }

  // Discards every staged scan on the session and hands it back fresh --
  // the "start over" escape hatch for a wrong box / mis-pick spree. Only
  // legal pre-complete (see ResetPickingSessionUseCase's doc); confirmed
  // via resetConfirmOpen since it's destructive to in-progress work.
  async function handleResetSession() {
    if (!session || !token) return
    setResetting(true)
    setError('')
    try {
      await pickingApi.resetPickingSession(token, session.session.id)
      setResetConfirmOpen(false)
      setPickLine(null)
      setAddItemOpen(null)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reset picking session')
    } finally {
      setResetting(false)
    }
  }

  // handleConfirmComplete is the ONLY place that calls the completePicking
  // API -- the header's "Complete" button just opens the review step
  // (setConfirmingFinish(true) below), it never calls this directly.
  // resolutions is only ever passed when resubmitting from
  // StockShortfallDialog; the first attempt always omits it.
  async function handleConfirmComplete(resolutions?: Record<string, StockShortfallResolution>) {
    if (!session || !token) return
    setCompleting(true)
    setError('')
    try {
      const result = await pickingApi.completePicking(token, session.session.id, resolutions)
      if (!result.completed) {
        setStockShortfalls(result.stock_shortfalls ?? [])
        return
      }
      setStockShortfalls(null)
      setCompleteResult(result)
      setConfirmingFinish(false)
      await loadOrders()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete picking')
    } finally {
      setCompleting(false)
    }
  }

  const pickedByLineItem = session ? pickedQuantityByLineItem(session.scans) : new Map<string, number>()
  const totalItems = session?.line_items.reduce((s, li) => s + li.quantity, 0) ?? 0
  const pickedItems = session ? session.line_items.reduce((s, li) => s + Math.min(li.quantity, pickedByLineItem.get(li.id) ?? 0), 0) : 0
  const progress = totalItems > 0 ? (pickedItems / totalItems) * 100 : 0
  const allPicked = totalItems > 0 && pickedItems >= totalItems

  const filteredOrders = orders.filter(o => {
    if (!search.trim()) return true
    const q = search.toLowerCase()
    return (o.number || o.external_order_id).toLowerCase().includes(q) || customerName(o).toLowerCase().includes(q)
  })

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500'

  return (
    <div className="flex h-full">
      {/* Order queue */}
      <div className="w-64 border-r border-line bg-sunken flex flex-col overflow-hidden shrink-0">
        <div className="px-4 py-3 border-b border-line">
          <p className="text-xs font-semibold text-ink">Orders to Pick</p>
          <p className="text-2xs text-ink-4 mt-0.5">{orders.length} unfulfilled</p>
        </div>
        <div className="px-3 py-2 border-b border-line">
          <div className="relative">
            <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search orders..."
              className="w-full pl-7 pr-2 py-1.5 text-2xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {ordersLoading && <p className="px-4 py-3 text-2xs text-ink-4">Loading…</p>}
          {ordersError && <p className="px-4 py-3 text-2xs text-danger">{ordersError}</p>}
          {!ordersLoading && filteredOrders.length === 0 && (
            <p className="px-4 py-3 text-2xs text-ink-4">No unfulfilled orders</p>
          )}
          {filteredOrders.map(o => (
            <button
              key={o.id}
              onClick={() => selectOrder(o)}
              className={clsx(
                'w-full text-left px-4 py-3 border-b border-line transition-colors',
                activeOrder?.id === o.id
                  ? 'dark:bg-blue-600/10 bg-blue-50 border-l-2 border-l-blue-500'
                  : 'hover:bg-raised',
              )}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs font-medium text-ink">#{o.number || o.external_order_id}</span>
                {activeOrder?.id === o.id && session?.session.status === 'in_progress' && (
                  <span className="text-2xs font-medium text-accent">In progress</span>
                )}
              </div>
              <p className="text-2xs text-ink-3 truncate">{customerName(o)}</p>
              <p className="text-2xs text-ink-4 mt-0.5">{formatMoney(o.total_minor_units, o.currency_code)}</p>
            </button>
          ))}
        </div>
      </div>

      {/* Main pick area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!activeOrder && !(urlSessionId && (sessionLoading || error)) && (
          <div className="flex-1 flex flex-col items-center justify-center text-ink-4">
            <ScanBarcode size={32} className="mb-3 opacity-40" />
            <p className="text-sm">Select an order to start picking</p>
          </div>
        )}

        {!session && !sessionLoading && error && <SessionErrorPanel error={error} onBack={backToOrders} />}

        {(activeOrder || urlSessionId) && sessionLoading && !session && (
          <div className="flex-1 flex items-center justify-center text-ink-4 text-sm">Loading…</div>
        )}

        {activeOrder && completeResult && (
          <div className="flex-1 flex flex-col items-center justify-center p-8">
            <CheckCircle size={40} className="text-success mb-3" />
            <p className="text-sm font-semibold text-ink">
              #{activeOrder.number || activeOrder.external_order_id} marked fulfilled
            </p>
            {!!completeResult.missing_items?.length && (
              <div className="mt-4 w-full max-w-sm rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-3">
                <p className="text-2xs font-medium text-warning mb-2">Not fully picked</p>
                {completeResult.missing_items.map(m => (
                  <p key={m.line_item_id} className="text-2xs text-ink-2">
                    {m.name}: {m.picked_qty}/{m.expected_qty}
                  </p>
                ))}
              </div>
            )}

            <button
              type="button" onClick={continueToPacking}
              className="mt-5 px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 flex items-center gap-1.5"
            >
              <PackageCheck size={13} /> Continue to Packing
            </button>

            <button
              type="button" onClick={backToOrders}
              className="mt-3 px-4 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
            >
              Back to orders
            </button>
          </div>
        )}

        {stockShortfalls && (
          <StockShortfallDialog
            shortfalls={stockShortfalls}
            submitting={completing}
            onCancel={() => setStockShortfalls(null)}
            onConfirm={resolutions => handleConfirmComplete(resolutions)}
          />
        )}

        {activeOrder && session && confirmingFinish && !completeResult && !stockShortfalls && (
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="flex items-center gap-3 px-6 py-4 border-b border-line shrink-0">
              <button onClick={() => setConfirmingFinish(false)} className="p-1 -ml-1" title="Back to picking">
                <ArrowLeft size={18} className="text-ink-3" />
              </button>
              <h2 className="text-sm font-semibold text-ink flex-1">Review before completing</h2>
            </div>
            {/* Customer/address stays visible here too, not just during
                active picking -- staff confirming completion should never
                have to go back just to double-check who this ships to. */}
            <div className="px-6 py-3 border-b border-line shrink-0">
              <p className="text-sm font-medium text-ink">
                #{activeOrder.number || activeOrder.external_order_id} · {customerName(activeOrder)}
              </p>
              {shippingAddressLine(activeOrder.shipping) && (
                <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(activeOrder.shipping)}</p>
              )}
              {(activeOrder.shipping_method_title || activeOrder.shipping_method) && (
                <p className="text-xs text-ink-4 mt-0.5">
                  Shipping: {activeOrder.shipping_method_title || activeOrder.shipping_method}
                </p>
              )}
            </div>
            <div className="flex-1 overflow-y-auto px-6 py-4 space-y-2">
              {(() => {
                const missing = computeMissingItems(session.line_items, session.scans)
                return missing.length === 0 ? (
                  <p className="text-sm text-ink-3">Every line is fully picked.</p>
                ) : (
                  <>
                    <p className="text-sm text-warning">{missing.length} line(s) aren't fully picked yet:</p>
                    {missing.map(m => (
                      <div key={m.line_item_id} className="rounded-lg bg-raised border border-line px-3 py-2">
                        <p className="text-sm text-ink">{m.name}</p>
                        <p className="text-xs text-ink-4">expected {m.expected_qty} · picked {m.picked_qty}</p>
                      </div>
                    ))}
                  </>
                )
              })()}
            </div>
            <div className="p-6 border-t border-line shrink-0 space-y-2">
              {error && <p className="text-sm text-danger">{error}</p>}
              <button
                onClick={() => handleConfirmComplete()} disabled={completing}
                className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
              >
                {completing ? 'Completing…' : 'Confirm & Complete'}
              </button>
              <button
                onClick={() => setConfirmingFinish(false)}
                className="w-full py-2.5 rounded-xl dark:bg-night-800 bg-gray-100 text-ink-2 text-sm font-medium"
              >
                Back to scanning
              </button>
            </div>
          </div>
        )}

        {activeOrder && session && !completeResult && !confirmingFinish && (
          <>
            {/* Header */}
            <div className={clsx(
              'flex items-center gap-4 px-6 py-3 border-b border-line shrink-0',
            )}>
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-0.5">
                  <h2 className="text-sm font-semibold text-ink">#{activeOrder.number || activeOrder.external_order_id}</h2>
                  <span className="text-xs text-ink-4">·</span>
                  <span className="text-xs text-ink-3">{customerName(activeOrder)}</span>
                </div>
                <p className="text-xs text-ink-4">{formatMoney(activeOrder.total_minor_units, activeOrder.currency_code)}</p>
                {shippingAddressLine(activeOrder.shipping) && (
                  <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(activeOrder.shipping)}</p>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button" onClick={handlePause} disabled={pausing}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50"
                >
                  <PauseCircle size={13} /> {pausing ? 'Pausing…' : 'Pause'}
                </button>
                <button
                  type="button" onClick={() => setResetConfirmOpen(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-danger hover:bg-hover"
                >
                  <Undo2 size={13} /> Start over
                </button>
                <button
                  type="button" onClick={() => setAddItemOpen({ skuId: '', productName: '' })}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover"
                >
                  <PackagePlus size={13} /> Add item
                </button>
                <button
                  type="button" onClick={() => setConfirmingFinish(true)}
                  className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700 disabled:opacity-50"
                >
                  <CheckCircle size={13} /> Complete
                </button>
              </div>
            </div>

            {/* Scan input */}
            <div className="px-6 py-4 border-b border-line shrink-0 dark:bg-night-900/80 bg-gray-50">
              <div className="flex-1 flex items-center gap-3 px-4 py-3 rounded-xl border-2 dark:border-blue-500 border-blue-400 bg-raised">
                <ScanBarcode size={20} className="text-accent" />
                <input
                  ref={codeRef}
                  type="text"
                  value={code}
                  onChange={e => setCode(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleScanSubmit()}
                  placeholder="Scan barcode or type and press Enter..."
                  disabled={busy || !!pickLine || !!addItemOpen || !!scanErrorAlert}
                  className="flex-1 bg-transparent text-sm text-ink outline-none focus-visible:ring-2 focus-visible:ring-focus/60 placeholder-ink-4 font-mono disabled:opacity-50"
                />
              </div>
              {error && <p className="text-2xs text-danger mt-2">{error}</p>}
            </div>

            {/* Progress */}
            <div className="px-6 py-3 border-b border-line shrink-0">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs text-ink-3">Progress</span>
                <span className="text-xs font-medium text-ink">{pickedItems} / {totalItems} items</span>
              </div>
              <div className="h-2 dark:bg-night-700 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className={clsx('h-full rounded-full transition-all duration-500', allPicked ? 'bg-green-500' : 'bg-blue-500')}
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>

            {/* Line items */}
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
              {session.line_items.map(li => {
                const picked = pickedByLineItem.get(li.id) ?? 0
                const done = picked >= li.quantity
                const partial = picked > 0 && picked < li.quantity
                return (
                  <div
                    key={li.id}
                    onClick={() => !done && openLineForPick(li, picked)}
                    className={clsx(
                      'flex items-center gap-4 p-3 rounded-xl border transition-all',
                      li.sku_id && !done && 'cursor-pointer',
                      done ? 'dark:bg-green-500/5 dark:border-green-500/20 border-green-200 bg-green-50' :
                      partial ? 'dark:bg-amber-500/5 dark:border-amber-500/20 border-amber-200 bg-amber-50' :
                      'bg-raised border-line',
                    )}
                  >
                    {done ? (
                      <div className="w-10 h-10 rounded-lg bg-green-500/20 flex items-center justify-center shrink-0">
                        <CheckCircle size={18} className="text-success" />
                      </div>
                    ) : (
                      <LineItemThumb token={token ?? ''} productId={li.product_id} />
                    )}
                    <div className="flex-1 min-w-0">
                      <p className={clsx('text-sm font-medium', done ? 'dark:text-green-300 text-success' : 'text-ink')}>{li.name}</p>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="text-2xs font-mono text-ink-4">{li.sku_code}</span>
                        {!li.sku_id && <span className="text-2xs text-warning">unlinked — scan to resolve</span>}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <p className={clsx('text-lg font-bold tabular-nums', done ? 'text-success' : partial ? 'text-warning' : 'text-ink')}>
                        {picked}/{li.quantity}
                      </p>
                      <p className="text-2xs text-ink-4">picked</p>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Staged scans log */}
            {session.scans.length > 0 && (
              <div className="border-t border-line px-4 py-2 max-h-32 overflow-y-auto shrink-0">
                <p className="text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1">This session</p>
                {session.scans.map(s => (
                  <div key={s.id} className="flex items-center justify-between py-0.5">
                    <span className="text-2xs text-ink-3">
                      {ACTION_LABEL[s.action_type]} · qty {s.quantity}
                    </span>
                    <button type="button" onClick={() => handleDeleteScan(s.id)} aria-label="Remove scan" className="text-ink-4 hover:text-danger">
                      <Trash2 size={11} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </div>

      {/* Pick / Substitute confirm panel */}
      {pickLine && (
        <div className="w-80 border-l border-line bg-sunken flex flex-col shrink-0">
          <div className="flex items-center justify-between px-4 py-3 border-b border-line">
            <p className="text-xs font-semibold text-ink">{pickLine.lineItem.name}</p>
            <button type="button" onClick={() => setPickLine(null)} aria-label="Close" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
              <X size={14} />
            </button>
          </div>
          <div className="p-4 space-y-4">
            <div className="flex items-center rounded-lg border border-line overflow-hidden">
              <button
                type="button" onClick={() => setPickPanelMode('pick')}
                className={clsx('flex-1 px-3 py-1.5 text-2xs font-medium', pickPanelMode === 'pick' ? 'bg-blue-600 text-white' : 'bg-raised text-ink-3')}
              >
                Pick
              </button>
              <button
                type="button" onClick={() => { setPickPanelMode('substitute'); setSubstituteRows([newSubstituteRow()]) }}
                className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', pickPanelMode === 'substitute' ? 'bg-blue-600 text-white' : 'bg-raised text-ink-3')}
              >
                <Repeat size={11} /> Substitute
              </button>
              <button
                type="button" onClick={() => setPickPanelMode('refund')}
                className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', pickPanelMode === 'refund' ? 'bg-blue-600 text-white' : 'bg-raised text-ink-3')}
              >
                <Undo2 size={11} /> Refund
              </button>
            </div>

            {pickPanelMode === 'refund' ? (
              <>
                <p className="text-2xs text-ink-3">
                  Can't fulfill this and don't want to substitute? Refund part or all of this line's own quantity
                  ({pickLine.lineItem.quantity}) instead -- money only moves once the session is completed.
                </p>
                <label className="block">
                  <span className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity to refund</span>
                  <input
                    type="number" min={1} max={pickLine.lineItem.quantity}
                    value={refundQty} onChange={e => setRefundQty(e.target.value)}
                    className={inputClass}
                  />
                </label>
                <label className="block">
                  <span className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Reason (optional)</span>
                  <input
                    type="text" value={refundReason} onChange={e => setRefundReason(e.target.value)}
                    className={inputClass}
                  />
                </label>
                <button
                  type="button" onClick={confirmRefund} disabled={busy || !refundReady}
                  className="w-full px-3 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50"
                >
                  Confirm refund
                </button>
              </>
            ) : pickPanelMode === 'pick' ? (
              <>
                <div>
                  <label className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity</label>
                  <input
                    type="number" min={1} max={pickLine.lineItem.quantity - pickLine.pickedSoFar || undefined}
                    value={pickQty} onChange={e => setPickQty(Math.max(1, parseInt(e.target.value, 10) || 1))}
                    className={inputClass}
                  />
                </div>
                <button
                  type="button" onClick={confirmPick} disabled={busy}
                  className="w-full px-3 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50"
                >
                  Confirm pick
                </button>
              </>
            ) : (
              <>
                <p className="text-2xs text-ink-3">
                  Replace with one or more products -- each row's "covers" quantity must add up to this line's
                  remaining quantity ({substituteRemaining}) before you can confirm.
                </p>
                {substituteRows.map((row, i) => (
                  <div key={row.key} className="space-y-2 rounded-lg border border-line p-2.5">
                    <input
                      type="text" value={row.code} onChange={e => updateSubstituteRow(row.key, { code: e.target.value })}
                      placeholder="Scan or type replacement SKU/barcode"
                      className={inputClass} autoFocus={i === 0}
                    />
                    <div className="grid grid-cols-2 gap-2">
                      <label className="block">
                        <span className="block text-2xs text-ink-4 mb-1">Qty added</span>
                        <input
                          type="number" min={1} value={row.substituteQty}
                          onChange={e => updateSubstituteRow(row.key, { substituteQty: e.target.value })}
                          className={inputClass}
                        />
                      </label>
                      <label className="block">
                        <span className="block text-2xs text-ink-4 mb-1">Covers (of original)</span>
                        <input
                          type="number" min={1} value={row.coveredQty}
                          onChange={e => updateSubstituteRow(row.key, { coveredQty: e.target.value })}
                          className={inputClass}
                        />
                      </label>
                    </div>
                    <label className="flex items-center gap-2 text-2xs text-ink-3">
                      <input
                        type="checkbox" checked={row.returnToStock}
                        onChange={e => updateSubstituteRow(row.key, { returnToStock: e.target.checked })}
                      />
                      Return original quantity to stock
                    </label>
                    {substituteRows.length > 1 && (
                      <button
                        type="button" onClick={() => setSubstituteRows(rows => rows.filter((_, idx) => idx !== i))}
                        className="text-2xs text-ink-4 hover:text-danger"
                      >
                        Remove this row
                      </button>
                    )}
                  </div>
                ))}
                <button
                  type="button" onClick={() => setSubstituteRows(rows => [...rows, newSubstituteRow()])}
                  className="w-full py-2 rounded-lg border border-dashed dark:border-night-700 border-gray-300 text-2xs text-ink-3"
                >
                  + Add another substitute
                </button>
                <p className="text-2xs text-center text-ink-4">
                  Covers {substituteCoveredTotal} of {substituteRemaining}
                </p>
                <button
                  type="button" onClick={confirmSubstitute} disabled={busy || !substituteReady}
                  className="w-full px-3 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50"
                >
                  Confirm substitute
                </button>
              </>
            )}
          </div>
        </div>
      )}

      {/* Add item (freebie / new item) panel */}
      {addItemOpen && (
        <div className="w-80 border-l border-line bg-sunken flex flex-col shrink-0">
          <div className="flex items-center justify-between px-4 py-3 border-b border-line">
            <p className="text-xs font-semibold text-ink">Add item not on order</p>
            <button type="button" onClick={() => setAddItemOpen(null)} aria-label="Close" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
              <X size={14} />
            </button>
          </div>
          <div className="p-4 space-y-4">
            {!addItemOpen.skuId ? (
              <>
                <p className="text-2xs text-ink-3">Scan or type the product's SKU/barcode.</p>
                <input
                  type="text" value={code} onChange={e => setCode(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleAddItemScanSubmit()}
                  placeholder="Scan or type SKU/barcode" className={inputClass} autoFocus
                />
              </>
            ) : (
              <>
                <p className="text-xs text-ink">{addItemOpen.productName}</p>
                <div className="flex items-center rounded-lg border border-line overflow-hidden">
                  <button
                    type="button" onClick={() => setAddItemFreebie(true)}
                    className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', addItemFreebie ? 'bg-blue-600 text-white' : 'bg-raised text-ink-3')}
                  >
                    <Gift size={11} /> Freebie
                  </button>
                  <button
                    type="button" onClick={() => setAddItemFreebie(false)}
                    className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', !addItemFreebie ? 'bg-blue-600 text-white' : 'bg-raised text-ink-3')}
                  >
                    <PackagePlus size={11} /> New item
                  </button>
                </div>
                <div>
                  <label className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity</label>
                  <input
                    type="number" min={1} value={addItemQty}
                    onChange={e => setAddItemQty(Math.max(1, parseInt(e.target.value, 10) || 1))}
                    className={inputClass}
                  />
                </div>
                {!addItemFreebie && (
                  <div>
                    <label className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Price</label>
                    <input
                      type="number" min={0} step="0.01" value={addItemPrice}
                      onChange={e => setAddItemPrice(e.target.value)}
                      className={inputClass}
                    />
                  </div>
                )}
                <button
                  type="button" onClick={confirmAddItem} disabled={busy}
                  className="w-full px-3 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50"
                >
                  Add to order
                </button>
              </>
            )}
          </div>
        </div>
      )}

      {scanErrorAlert && (
        <Dialog onClose={dismissScanError} label="Scan error" initialFocus="dialog" className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-xs rounded-xl bg-surface border dark:border-red-500/40 border-red-300 p-5 text-center shadow-xl">
            <p className="text-sm font-semibold text-danger mb-4">{scanErrorAlert}</p>
            <button
              type="button" onClick={dismissScanError}
              className="w-full py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700"
            >
              OK
            </button>
          </div>
        </Dialog>
      )}

      {resetConfirmOpen && (
        <Dialog onClose={() => setResetConfirmOpen(false)} label="Start over?" dismissible={!resetting} className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-xs rounded-xl bg-surface border border-line p-5 text-center shadow-xl">
            <p className="text-sm font-semibold text-ink mb-1">Start over?</p>
            <p className="text-xs text-ink-3 mb-4">
              This discards every item picked so far on this order and starts the session fresh. This can't be undone.
            </p>
            <div className="flex gap-2">
              <button
                type="button" onClick={() => setResetConfirmOpen(false)} disabled={resetting}
                className="flex-1 py-2 rounded-lg border border-line text-xs font-medium text-ink-2 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button" onClick={handleResetSession} disabled={resetting}
                className="flex-1 py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700 disabled:opacity-50"
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
