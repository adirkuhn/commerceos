import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ScanBarcode, CheckCircle, Package, X, Trash2, ArrowLeft, Undo2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useRoutedSession } from '../oms/useRoutedSession'
import { SessionErrorPanel } from '../oms/SessionErrorPanel'
import { PackingOrderQueue } from '../packing/PackingOrderQueue'
import { PickListPrintView } from '../packing/PickListPrintView'
import { customerName, formatMoney } from '../packing/orderDisplay'
import { canTakeOverLocks } from '../packing/lockDisplay'
import { usePackingQueue } from '../oms/hooks/usePackingQueue'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { useQueryString } from '../routing/useQueryState'
import type { SalesOrder, Address } from '../oms/types'
import * as packingApi from '../scanner/api'
import type { PackingSessionDetail, PickingSessionLineItem, PackScan, CompletePackingResult } from '../scanner/types'
import { packedQuantityByLineItem, computeMissingPackItems } from '../scanner/packingProgress'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { useConnections } from '../integrations/hooks/useConnections'
import * as shippingApi from '../shipping/api'
import type { RecipientAddress } from '../shipping/types'
import { guessCarrierFromShippingMethod } from '../shipping/carrierGuess'
import { AddressConfirmModal } from '../shipping/AddressConfirmModal'
import { LabelPreviewModal } from '../shipping/LabelPreviewModal'
import { ExistingShipmentConfirmModal } from '../shipping/ExistingShipmentConfirmModal'
import type { Shipment } from '../shipping/types'
import { loadShipmentLabelIntoTab, openShipmentLabel } from '../shipping/openLabel'
import { LabelOpenedNotice } from '../shipping/LabelOpenedNotice'
import { playErrorSound } from '../scanner/errorSound'
import { Dialog } from '../components/common/Dialog'

// Duplicated from Picking.tsx on purpose -- same small-helper convention
// used elsewhere in this codebase rather than a shared cross-screen import.
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

function shippingAddressLine(address?: Address): string {
  if (!address) return ''
  const streetAndLocality = [address.address_1, address.address_2, [address.postcode, address.city].filter(Boolean).join(' ')]
    .filter(Boolean).join(', ')
  return [streetAndLocality, address.country].filter(Boolean).join(', ')
}

const PACKING_PATH = '/packing'

// Each open session has its own URL (/packing/:sessionId) -- see
// useRoutedSession. ?order=<id> is kept as an action URL (Picking's
// "Continue to packing"): it starts/resumes that order's session, then
// replaces itself with the session URL.
export default function Packing() {
  const { token, user } = useAuth()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  // ?pick=<id,id> shows the multi-order pick list. Opening pushes (Back
  // closes it, refresh keeps it); the Close button replaces, so closing
  // doesn't leave a Back step that reopens it.
  const [pickParam, setPickParam] = useQueryString('pick', { history: 'push' })
  const [, clearPickParam] = useQueryString('pick')
  const pickIds = pickParam ? pickParam.split(',').filter(Boolean) : []

  const queue = usePackingQueue(token ?? '')
  const nameFor = useMemberNames(token ?? '')

  const [activeOrder, setActiveOrder] = useState<SalesOrder | null>(null)
  const [session, setSession] = useState<PackingSessionDetail | null>(null)
  const [sessionLoading, setSessionLoading] = useState(false)
  const [error, setError] = useState('')

  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)
  const codeRef = useRef<HTMLInputElement>(null)

  const [packLine, setPackLine] = useState<{ lineItem: PickingSessionLineItem; packedSoFar: number } | null>(null)
  const [packQty, setPackQty] = useState(1)

  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const [resetting, setResetting] = useState(false)

  const [completing, setCompleting] = useState(false)
  const [completeResult, setCompleteResult] = useState<CompletePackingResult | null>(null)
  const [confirmingFinish, setConfirmingFinish] = useState(false)

  const { connections } = useConnections(token ?? '')
  const myParcelConnection = connections.find(c => c.provider === 'myparcel')
  // Same 3-step label flow as Picking.tsx used to have -- moved here now
  // that packing (not picking) is what gates "ready to ship".
  const [labelStep, setLabelStep] = useState<'address' | 'preview' | null>(null)
  const [pendingRecipient, setPendingRecipient] = useState<RecipientAddress | null>(null)
  const [pendingCarrier, setPendingCarrier] = useState<number | null>(null)
  const [pendingPackageType, setPendingPackageType] = useState<number | null>(null)
  const [labelSize, setLabelSize] = useState<'A4' | 'A6'>('A4')
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null)
  const [labelTab, setLabelTab] = useState<Window | null>(null)
  const [existingShipment, setExistingShipment] = useState<Shipment | null>(null)
  const [autoPrintEnabled, setAutoPrintEnabled] = useState(false)
  const [connectionDefaultCarrier, setConnectionDefaultCarrier] = useState(1)
  const [connectionDefaultPackageType, setConnectionDefaultPackageType] = useState(1)
  useEffect(() => {
    if (!token || !myParcelConnection) return
    shippingApi.getShippingSettings(token, myParcelConnection.id)
      .then(result => {
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
        setConnectionDefaultCarrier(result.settings?.default_carrier ?? 1)
        setConnectionDefaultPackageType(result.settings?.default_package_type ?? 1)
      })
      .catch(() => setAutoPrintEnabled(false))
  }, [token, myParcelConnection])

  useEffect(() => {
    setLabelStep(null)
    setPendingRecipient(null)
    setPendingCarrier(null)
    setPendingPackageType(null)
    setCreatedShipmentId(null)
    setLabelTab(null)
    setExistingShipment(null)
  }, [activeOrder?.id])


  const loadSession = useCallback(async (sessionId: string) => {
    if (!token) return
    try {
      setSession(await packingApi.getPackingSession(token, sessionId))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load packing session')
    } finally {
      setSessionLoading(false)
    }
  }, [token])

  useEffect(() => { codeRef.current?.focus() }, [session, packLine])

  // Heartbeat -- see Picking.tsx's identical mechanism for the doc.
  useEffect(() => {
    const sessionId = session?.session.id
    if (!sessionId || !token || session?.session.status === 'completed') return
    const interval = setInterval(() => {
      packingApi.touchPackingSession(token, sessionId).catch(err => {
        if (err instanceof Error && /locked/i.test(err.message)) {
          setError('Another user has taken over this packing session')
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
    setError('')
    setPackLine(null)
    setCreatedShipmentId(null)
    setLabelStep(null)
    setPendingRecipient(null)
    setConfirmingFinish(false)
    setExistingShipment(null)
  }

  const { urlSessionId, showSession, backToList: backToOrders } = useRoutedSession({
    basePath: PACKING_PATH,
    token,
    loadDetail: packingApi.getPackingSession,
    currentOrder: activeOrder,
    onLoading: () => { setSessionLoading(true); setError('') },
    onLoaded: (order, detail) => { setActiveOrder(order); setSession(detail); setSessionLoading(false) },
    onError: message => { setError(message); setSessionLoading(false) },
    onCleared: clearSelection,
  })

  async function selectOrder(order: SalesOrder) {
    setError('')
    setCompleteResult(null)
    setPackLine(null)
    setActiveOrder(order)
    setSession(null)
    setSessionLoading(true)
    try {
      const newSession = await packingApi.startOrResumePacking(token ?? '', order.id)
      if (showSession(newSession.id)) await loadSession(newSession.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start packing')
      setSessionLoading(false)
    }
  }

  // Action URL: Picking.tsx's "Continue to packing" lands here with
  // ?order=<id> so a picker who just finished goes straight into packing
  // the same order. Replaced (not pushed) by the session URL, so Back
  // doesn't re-trigger the start.
  const orderToStart = searchParams.get('order')
  useEffect(() => {
    if (!orderToStart || !token) return
    let current = true
    setSessionLoading(true)
    packingApi.startOrResumePacking(token, orderToStart)
      .then(started => {
        if (current) navigate(`${PACKING_PATH}/${encodeURIComponent(started.id)}`, { replace: true })
      })
      .catch(err => {
        if (!current) return
        setError(err instanceof Error ? err.message : 'Failed to start packing')
        setSessionLoading(false)
      })
    return () => { current = false }
  }, [orderToStart, token, navigate])

  // Same "instant increment on scan, blocking modal on any mis-scan"
  // pattern as Picking.tsx's handleScanSubmit -- see the fix note there for
  // why a mis-scan must never fall through to a non-blocking inline error.
  async function handleScanSubmit() {
    const scannedCode = code.trim()
    if (!scannedCode || !session || !token || scanErrorAlert) return
    setError('')
    setBusy(true)
    try {
      await packingApi.scanPackItem(token, session.session.id, { code: scannedCode, quantity: 1 })
      setCode('')
      await loadSession(session.session.id)
    } catch (err) {
      playErrorSound()
      setScanErrorAlert(err instanceof Error ? err.message : 'Item not on this order')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeRef.current?.focus()
  }

  function openLineForPack(li: PickingSessionLineItem, packedSoFar: number) {
    if (!li.sku_id) return
    setPackLine({ lineItem: li, packedSoFar })
    setPackQty(Math.max(1, li.quantity - packedSoFar))
  }

  async function confirmPack() {
    if (!packLine || !session || !token || !packLine.lineItem.sku_id) return
    setBusy(true)
    setError('')
    try {
      await packingApi.scanPackItem(token, session.session.id, { sku_id: packLine.lineItem.sku_id, quantity: packQty })
      setPackLine(null)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record pack')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
  }

  async function handleDeleteScan(scanId: string) {
    if (!session || !token) return
    setError('')
    try {
      await packingApi.deletePackScan(token, session.session.id, scanId)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove scan')
    }
  }

  // Discards every pack scan on the session and hands it back fresh -- the
  // "start over" escape hatch for switching boxes or a mis-scan spree. Only
  // legal pre-complete (see ResetPackingSessionUseCase's doc).
  async function handleResetSession() {
    if (!session || !token) return
    setResetting(true)
    setError('')
    try {
      await packingApi.resetPackingSession(token, session.session.id)
      setResetConfirmOpen(false)
      setPackLine(null)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reset packing session')
    } finally {
      setResetting(false)
    }
  }

  async function handleConfirmComplete() {
    if (!session || !token) return
    setCompleting(true)
    setError('')
    try {
      const result = await packingApi.completePacking(token, session.session.id)
      setCompleteResult(result)
      setConfirmingFinish(false)
      await queue.refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete packing')
    } finally {
      setCompleting(false)
    }
  }

  async function handleCreateLabelClick() {
    if (!token || !activeOrder) return
    const shipments = await shippingApi.listShipmentsForOrder(token, activeOrder.id)
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  async function handleOpenLabel(shipmentId: string) {
    if (!token) return
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  async function handlePrintLabel() {
    if (!token || !activeOrder || !myParcelConnection || !pendingRecipient) return
    const tab = window.open('', '_blank')
    let shipment
    try {
      shipment = await shippingApi.createShipment(token, {
        carrier: pendingCarrier ?? connectionDefaultCarrier,
        package_type: pendingPackageType ?? connectionDefaultPackageType,
        recipient: pendingRecipient,
        connection_id: myParcelConnection.id, sales_order_id: activeOrder.id,
      })
    } catch (err) {
      tab?.close()
      throw err
    }
    setLabelStep(null)
    setCreatedShipmentId(shipment.id)
    try {
      setLabelTab(await loadShipmentLabelIntoTab(tab, token, shipment.id, labelSize))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  const packedByLineItem = session ? packedQuantityByLineItem(session.scans) : new Map<string, number>()
  const totalItems = session?.line_items.reduce((s, li) => s + li.quantity, 0) ?? 0
  const packedItems = session ? session.line_items.reduce((s, li) => s + Math.min(li.quantity, packedByLineItem.get(li.id) ?? 0), 0) : 0
  const progress = totalItems > 0 ? (packedItems / totalItems) * 100 : 0
  const allPacked = totalItems > 0 && packedItems >= totalItems

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'


  if (token && pickIds.length > 0) {
    return <PickListPrintView token={token} orderIds={pickIds} onClose={() => clearPickParam(null)} />
  }

  return (
    <div className="flex h-full">
      <PackingOrderQueue
        entries={queue.entries} loading={queue.loading} error={queue.error}
        currentUserId={user?.user_id ?? ''} isAdmin={canTakeOverLocks(user?.role)} nameFor={nameFor}
        activeOrderId={activeOrder?.id ?? null} activeInProgress={session?.session.status === 'in_progress'}
        onSelect={selectOrder} onOpenPickList={ids => setPickParam(ids.join(','))}
      />

      {/* Main pack area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!activeOrder && !((urlSessionId || orderToStart) && (sessionLoading || error)) && (
          <div className="flex-1 flex flex-col items-center justify-center text-ink-4">
            <Package size={32} className="mb-3 opacity-40" />
            <p className="text-sm">Select an order to start packing</p>
          </div>
        )}

        {!session && !sessionLoading && error && <SessionErrorPanel error={error} onBack={backToOrders} />}

        {(activeOrder || urlSessionId || orderToStart) && sessionLoading && !session && (
          <div className="flex-1 flex items-center justify-center text-ink-4 text-sm">Loading…</div>
        )}

        {activeOrder && completeResult && (
          <div className="flex-1 flex flex-col items-center justify-center p-8">
            <CheckCircle size={40} className="text-success mb-3" />
            <p className="text-sm font-semibold text-ink">
              #{activeOrder.number || activeOrder.external_order_id} packed
            </p>
            {!!completeResult.missing_items?.length && (
              <div className="mt-4 w-full max-w-sm rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-3">
                <p className="text-2xs font-medium text-warning mb-2">Not fully packed</p>
                {completeResult.missing_items.map(m => (
                  <p key={m.line_item_id} className="text-2xs text-ink-2">
                    {m.name}: {m.packed_qty}/{m.expected_qty}
                  </p>
                ))}
              </div>
            )}

            {myParcelConnection && !createdShipmentId && (
              <button
                type="button" onClick={handleCreateLabelClick}
                className="mt-5 px-4 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
              >
                Create shipping label
              </button>
            )}
            {createdShipmentId && (
              <button
                type="button" onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
                className="mt-5 px-4 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
              >
                {autoPrintEnabled ? 'Retry print' : 'Print shipping label'}
              </button>
            )}

            <button
              type="button" onClick={backToOrders}
              className="mt-3 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover"
            >
              Back to orders
            </button>
          </div>
        )}

        {labelStep === 'address' && activeOrder && (
          <AddressConfirmModal
            token={token ?? ''}
            initialAddress={activeOrder.shipping}
            fallbackEmail={activeOrder.billing?.email}
            connectionId={myParcelConnection?.id}
            initialRecipient={pendingRecipient ?? undefined}
            initialCarrier={pendingCarrier ?? undefined}
            initialPackageType={pendingPackageType ?? undefined}
            defaultCarrier={guessCarrierFromShippingMethod(activeOrder.shipping_method_title, activeOrder.shipping_method, connectionDefaultCarrier)}
            defaultPackageType={connectionDefaultPackageType}
            onNext={({ carrier, packageType, recipient }) => { setPendingCarrier(carrier); setPendingPackageType(packageType); setPendingRecipient(recipient); setLabelStep('preview') }}
            onClose={() => setLabelStep(null)}
          />
        )}
        {labelStep === 'preview' && pendingRecipient && (
          <LabelPreviewModal
            recipient={pendingRecipient}
            carrier={pendingCarrier ?? connectionDefaultCarrier}
            packageType={pendingPackageType ?? connectionDefaultPackageType}
            labelSize={labelSize}
            onLabelSizeChange={setLabelSize}
            onBack={() => setLabelStep('address')}
            onPrint={handlePrintLabel}
            onClose={() => setLabelStep(null)}
          />
        )}
        {labelTab && (
          <LabelOpenedNotice onPrint={() => labelTab.print()} onDismiss={() => setLabelTab(null)} />
        )}
        {existingShipment && (
          <ExistingShipmentConfirmModal
            token={token ?? ''}
            shipment={existingShipment}
            onCancel={() => setExistingShipment(null)}
            onReprint={() => {
              setCreatedShipmentId(existingShipment.id)
              handleOpenLabel(existingShipment.id)
              setExistingShipment(null)
            }}
            onCreateNew={() => {
              setExistingShipment(null)
              setLabelStep('address')
            }}
          />
        )}

        {activeOrder && session && confirmingFinish && !completeResult && (
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="flex items-center gap-3 px-6 py-4 border-b border-line shrink-0">
              <button aria-label="Back to packing" onClick={() => setConfirmingFinish(false)} className="p-1 -ml-1" title="Back to packing">
                <ArrowLeft size={18} className="text-ink-3" />
              </button>
              <h2 className="text-sm font-semibold text-ink flex-1">Review before completing</h2>
            </div>
            <div className="px-6 py-3 border-b border-line shrink-0">
              <p className="text-sm font-medium text-ink">
                #{activeOrder.number || activeOrder.external_order_id} · {customerName(activeOrder)}
              </p>
              {shippingAddressLine(activeOrder.shipping) && (
                <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(activeOrder.shipping)}</p>
              )}
            </div>
            <div className="flex-1 overflow-y-auto px-6 py-4 space-y-2">
              {(() => {
                const missing = computeMissingPackItems(session.line_items, session.scans)
                return missing.length === 0 ? (
                  <p className="text-sm text-ink-3">Every line is fully packed.</p>
                ) : (
                  <>
                    <p className="text-sm text-warning">{missing.length} line(s) aren't fully packed yet:</p>
                    {missing.map(m => (
                      <div key={m.line_item_id} className="rounded-lg bg-raised border border-line px-3 py-2">
                        <p className="text-sm text-ink">{m.name}</p>
                        <p className="text-xs text-ink-4">expected {m.expected_qty} · packed {m.packed_qty}</p>
                      </div>
                    ))}
                  </>
                )
              })()}
            </div>
            <div className="p-6 border-t border-line shrink-0 space-y-2">
              {error && <p className="text-sm text-danger">{error}</p>}
              <button
                onClick={handleConfirmComplete} disabled={completing}
                className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
              >
                {completing ? 'Completing…' : 'Confirm & complete'}
              </button>
              <button
                onClick={() => setConfirmingFinish(false)}
                className="w-full py-2.5 rounded-xl dark:bg-night-800 bg-gray-100 text-ink-2 text-sm font-medium"
              >
                Back to packing
              </button>
            </div>
          </div>
        )}

        {activeOrder && session && !completeResult && !confirmingFinish && (
          <>
            {/* Header */}
            <div className="flex items-center gap-4 px-6 py-3 border-b border-line shrink-0">
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
                  type="button" onClick={() => setResetConfirmOpen(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-danger hover:bg-hover"
                >
                  <Undo2 size={13} /> Start over
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
              <div className="flex-1 flex items-center gap-3 px-4 py-3 rounded-xl border-2  border-accent bg-raised">
                <ScanBarcode size={20} className="text-accent" />
                <input
                  ref={codeRef}
                  type="text"
                  value={code}
                  onChange={e => setCode(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleScanSubmit()}
                  placeholder="Scan barcode or type and press Enter..."
                  disabled={busy || !!packLine || !!scanErrorAlert}
                  className="flex-1 bg-transparent text-sm text-ink outline-none focus-visible:ring-2 focus-visible:ring-focus/60 placeholder-ink-4 font-mono disabled:opacity-50"
                />
              </div>
              {error && <p className="text-2xs text-danger mt-2">{error}</p>}
            </div>

            {/* Progress */}
            <div className="px-6 py-3 border-b border-line shrink-0">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs text-ink-3">Progress</span>
                <span className="text-xs font-medium text-ink">{packedItems} / {totalItems} items</span>
              </div>
              <div className="h-2 dark:bg-night-700 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className={clsx('h-full rounded-full transition-all duration-500', allPacked ? 'bg-green-500' : 'bg-accent-solid')}
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>

            {/* Line items */}
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
              {session.line_items.map(li => {
                const packed = packedByLineItem.get(li.id) ?? 0
                const done = packed >= li.quantity
                const partial = packed > 0 && packed < li.quantity
                return (
                  <div
                    key={li.id}
                    onClick={() => !done && openLineForPack(li, packed)}
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
                      <span className="text-2xs font-mono text-ink-4">{li.sku_code}</span>
                    </div>
                    <div className="text-right shrink-0">
                      <p className={clsx('text-lg font-bold tabular-nums', done ? 'text-success' : partial ? 'text-warning' : 'text-ink')}>
                        {packed}/{li.quantity}
                      </p>
                      <p className="text-2xs text-ink-4">packed</p>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Staged scans log */}
            {session.scans.length > 0 && (
              <div className="border-t border-line px-4 py-2 max-h-32 overflow-y-auto shrink-0">
                <p className="text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1">This session</p>
                {session.scans.map((s: PackScan) => (
                  <div key={s.id} className="flex items-center justify-between py-0.5">
                    <span className="text-2xs text-ink-3">Packed · qty {s.quantity}</span>
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

      {/* Pack confirm panel */}
      {packLine && (
        <div className="w-80 border-l border-line bg-sunken flex flex-col shrink-0">
          <div className="flex items-center justify-between px-4 py-3 border-b border-line">
            <p className="text-xs font-semibold text-ink">{packLine.lineItem.name}</p>
            <button type="button" onClick={() => setPackLine(null)} aria-label="Close" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
              <X size={14} />
            </button>
          </div>
          <div className="p-4 space-y-4">
            <div>
              <label className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity</label>
              <input
                type="number" min={1} max={packLine.lineItem.quantity - packLine.packedSoFar || undefined}
                value={packQty} onChange={e => setPackQty(Math.max(1, parseInt(e.target.value, 10) || 1))}
                className={inputClass}
              />
            </div>
            <button
              type="button" onClick={confirmPack} disabled={busy}
              className="w-full px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
            >
              Confirm pack
            </button>
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
              This discards every item packed so far on this order and starts the session fresh. This can't be undone.
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
