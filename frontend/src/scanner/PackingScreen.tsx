import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, Camera, X, Undo2, Gift, Lock, Tag } from 'lucide-react'
import { useSessionLock } from '../fulfillment/useSessionLock'
import { readClaims } from '../auth/jwtClaims'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { canTakeOverLocks, formatLockAge } from '../packing/lockDisplay'
import * as api from './api'
import { CameraScanner } from './CameraScanner'
import { playErrorSound } from './errorSound'
import { useWakeLock } from './useWakeLock'
import { computeMissingPackItems, packedQuantityByLineItem } from './packingProgress'
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
import type { PackingSessionDetail, PickingSessionLineItem, CompletePackingResult, StockShortfall, StockShortfallResolution } from './types'
import { LineChangeSheet } from './LineChangeSheet'
import { AddItemSheet } from './AddItemSheet'
import { StockShortfallDialog } from './StockShortfallDialog'
import { packScanLabel } from '../packing/packScanLabel'
import type { SalesOrder } from '../oms/types'
import { Dialog } from '../components/common/Dialog'
import { ProductThumb, SCANNER_THUMB } from '../pms/ProductThumb'
import { customerName, shippingAddressLine } from '../oms/salesOrderDisplay'
import { isAuthError } from '../auth/isAuthError'
import { UNKNOWN_CUSTOMER_LABEL } from './scannerLabels'

interface PackingScreenProps {
  token: string
  sessionId: string
  order: SalesOrder
  onExit: () => void
  onAuthError: () => void
}

export function PackingScreen({ token, sessionId, order, onExit, onAuthError }: PackingScreenProps) {
  const [detail, setDetail] = useState<PackingSessionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [code, setCode] = useState('')
  const [scanning, setScanning] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)
  const [cameraOpen, setCameraOpen] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  const [result, setResult] = useState<CompletePackingResult | null>(null)
  const [packLine, setPackLine] = useState<{ lineItem: PickingSessionLineItem; packedSoFar: number } | null>(null)
  const [addItemOpen, setAddItemOpen] = useState(false)
  // Set when finishing comes back completed:false -- a substitute or extra
  // item is short on tracked stock (see StockShortfallDialog).
  const [stockShortfalls, setStockShortfalls] = useState<StockShortfall[] | null>(null)
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const [resetting, setResetting] = useState(false)
  const codeInputRef = useRef<HTMLInputElement>(null)

  const { connections } = useConnections(token)
  const myParcelConnection = connections.find(c => c.provider === 'myparcel')
  // Same 3-step label flow as PickingScreen used to have -- see Picking.tsx's
  // doc comment; moved here now that packing (not picking) gates shipping.
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
    if (!myParcelConnection) return
    shippingApi.getShippingSettings(token, myParcelConnection.id)
      .then(result => {
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
        setConnectionDefaultCarrier(result.settings?.default_carrier ?? 1)
        setConnectionDefaultPackageType(result.settings?.default_package_type ?? 1)
      })
      .catch(() => setAutoPrintEnabled(false))
  }, [token, myParcelConnection])

  async function handleCreateLabelClick() {
    if (!myParcelConnection) return
    // Before packing is finished, making the label is working the order --
    // take the lock first (the heartbeat claims it when free).
    if (!result) {
      try {
        await api.touchPackingSession(token, sessionId)
        await load()
      } catch (err) {
        if (isAuthError(err)) return onAuthError()
        setError(err instanceof Error ? err.message : 'Someone else is packing this order')
        return
      }
    }
    const shipments = await shippingApi.listShipmentsForOrder(token, order.id)
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  async function handleOpenLabel(shipmentId: string) {
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  async function handlePrintLabel() {
    if (!myParcelConnection || !pendingRecipient) return
    const tab = window.open('', '_blank')
    let shipment
    try {
      shipment = await shippingApi.createShipment(token, {
        carrier: pendingCarrier ?? connectionDefaultCarrier,
        package_type: pendingPackageType ?? connectionDefaultPackageType,
        recipient: pendingRecipient,
        connection_id: myParcelConnection.id, sales_order_id: order.id,
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

  const load = async () => {
    try {
      setDetail(await api.getPackingSession(token, sessionId))
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to load packing session')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [sessionId])
  useEffect(() => { codeInputRef.current?.focus() }, [detail])

  // Opening only views; the first action takes the lock. While we hold it
  // we heartbeat it, and leaving the order releases it.
  const me = readClaims(token)
  const nameFor = useMemberNames(token)
  const { heldByOther } = useSessionLock({
    sessionId, me: me?.userId ?? '',
    lockedBy: detail?.session.locked_by, lockedAt: detail?.session.locked_at,
    active: !!detail && detail.session.status !== 'completed',
    touch: id => api.touchPackingSession(token, id),
    release: id => api.releasePackingSessionLock(token, id),
    onLost: () => setError('Another user has taken over this packing session'),
  })
  const readOnly = heldByOther !== null

  async function handleTakeOver() {
    try {
      await api.takeOverPackingSession(token, sessionId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to take over')
    }
  }
  useWakeLock()

  // Runs one staged action (scan, substitute, refund, extra item) and
  // reloads the session.
  async function run(action: () => Promise<unknown>, failure: string): Promise<boolean> {
    setError('')
    setScanning(true)
    try {
      await action()
      await load()
      return true
    } catch (err) {
      if (isAuthError(err)) {
        onAuthError()
        return false
      }
      setError(err instanceof Error ? err.message : failure)
      return false
    } finally {
      setScanning(false)
      codeInputRef.current?.focus()
    }
  }

  function stage(input: Parameters<typeof api.scanPackItem>[2]): Promise<boolean> {
    return run(() => api.scanPackItem(token, sessionId, input), 'Scan failed')
  }

  // Unlike picking's two-step resolve-then-stage, scanPackItem resolves and
  // records in one call (see ScanPackItemUseCase) -- any failure (SKU not on
  // this order, quantity would exceed ordered, barcode unrecognized) must
  // block the same way picking's mis-scans do, not fall through silently.
  async function handleScan(e: React.FormEvent) {
    e.preventDefault()
    const scannedCode = code.trim()
    if (!scannedCode || scanErrorAlert) return
    setCode('')
    const ok = await stage({ code: scannedCode, quantity: 1 })
    if (!ok) {
      playErrorSound()
      setScanErrorAlert(error || 'Item not on this order')
    }
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    setCode(text)
    stage({ code: text, quantity: 1 }).then(ok => {
      setCode('')
      if (!ok) {
        playErrorSound()
        setScanErrorAlert('Item not on this order')
      }
    })
  }

  async function handleDeleteScan(scanId: string) {
    try {
      await api.deletePackScan(token, sessionId, scanId)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to delete scan')
    }
  }

  async function handleResetSession() {
    setResetting(true)
    try {
      await api.resetPackingSession(token, sessionId)
      setResetConfirmOpen(false)
      setPackLine(null)
      setAddItemOpen(false)
      await load()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to reset packing session')
    } finally {
      setResetting(false)
    }
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeInputRef.current?.focus()
  }

  // resolutions is only passed when resubmitting from StockShortfallDialog.
  async function handleConfirmComplete(resolutions?: Record<string, StockShortfallResolution>) {
    setFinishing(true)
    try {
      const r = await api.completePacking(token, sessionId, resolutions)
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

  // The label flow runs both before finishing (from the working screen)
  // and after (from the done screen).
  // The label button shared by the review and done screens: create one, or
  // print the one just made.
  const labelButtonClass = 'w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-200 text-sm font-medium hover:border-accent transition-colors'
  const labelButton = createdShipmentId ? (
    <button type="button" onClick={() => handleOpenLabel(createdShipmentId)} className={labelButtonClass}>
      {autoPrintEnabled ? 'Retry print' : 'Print shipping label'}
    </button>
  ) : myParcelConnection && (
    <button type="button" onClick={handleCreateLabelClick} className={labelButtonClass}>
      Create shipping label
    </button>
  )

  const labelModals = (
    <>
        {labelStep === 'address' && (
        <AddressConfirmModal
          token={token}
          initialAddress={order.shipping}
          fallbackEmail={order.billing?.email}
          connectionId={myParcelConnection?.id}
          initialRecipient={pendingRecipient ?? undefined}
          initialCarrier={pendingCarrier ?? undefined}
          initialPackageType={pendingPackageType ?? undefined}
          defaultCarrier={guessCarrierFromShippingMethod(order.shipping_method_title, order.shipping_method, connectionDefaultCarrier)}
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
          token={token}
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
    </>
  )

  if (confirmingFinish && detail) {
    const missing = computeMissingPackItems(detail.line_items, detail.scans)
    return (
      <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
          <button aria-label="Back to packing" onClick={() => setConfirmingFinish(false)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to packing">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
        </div>
        <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
          <div className="flex items-baseline gap-1.5 flex-wrap">
            <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
            <span className="text-gray-400">·</span>
            <span className="text-base text-gray-300">{customerName(order, UNKNOWN_CUSTOMER_LABEL)}</span>
          </div>
          {shippingAddressLine(order.shipping) && (
            <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
          )}
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
          {missing.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-8">Every line is fully packed.</p>
          ) : (
            <>
              <p className="text-sm text-amber-400">{missing.length} line(s) aren't fully packed yet:</p>
              {missing.map(m => (
                <div key={m.line_item_id} className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                  <p className="text-sm text-gray-200">{m.name}</p>
                  <p className="text-xs text-gray-400">expected {m.expected_qty} · packed {m.packed_qty}</p>
                </div>
              ))}
            </>
          )}
        </div>
        <div className="p-4 border-t border-night-700 shrink-0 space-y-2">
          {error && <p className="text-sm text-red-400">{error}</p>}
          {labelButton}
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
        {stockShortfalls && (
          <StockShortfallDialog
            shortfalls={stockShortfalls} submitting={finishing}
            onConfirm={resolutions => handleConfirmComplete(resolutions)} onCancel={() => setStockShortfalls(null)}
          />
        )}
        {labelModals}
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
          <h1 className="text-lg font-semibold">Order packed</h1>
          {!result.missing_items?.length && <p className="text-sm text-gray-400">Every line was fully packed.</p>}
        </div>
        {!!result.missing_items?.length && (
          <div className="flex-1 overflow-y-auto px-6 pb-4 space-y-2">
            <p className="text-sm text-amber-400">{result.missing_items.length} line(s) weren't fully packed:</p>
            {result.missing_items.map(m => (
              <div key={m.line_item_id} className="rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <p className="text-sm text-gray-200">{m.name}</p>
                <p className="text-xs text-gray-400">expected {m.expected_qty} · packed {m.packed_qty}</p>
              </div>
            ))}
          </div>
        )}
        <div className="p-6 shrink-0 space-y-2">
          {labelButton}
          <button onClick={onExit} className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover">
            Done
          </button>
        </div>

        {labelModals}
      </div>
    )
  }

  const packedByLineItem = packedQuantityByLineItem(detail?.scans ?? [])
  const extras = (detail?.scans ?? []).filter(s => s.action_type === 'freebie' || s.action_type === 'new_item')

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label="Back" onClick={onExit} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Packing</h1>
        {myParcelConnection && !createdShipmentId && (
          <button aria-label="Create label"
            type="button" onClick={handleCreateLabelClick} disabled={readOnly} title="Create label"
            className="min-h-11 px-3 -my-2 flex items-center gap-1.5 rounded-lg bg-night-800 border border-night-700 text-sm font-medium text-gray-200 disabled:opacity-50"
          >
            <Tag size={16} /> Label
          </button>
        )}
        <button
          onClick={() => setConfirmingFinish(true)} disabled={readOnly}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          Finish
        </button>
      </div>

      <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
        <div className="flex items-baseline gap-1.5 flex-wrap">
          <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
          <span className="text-gray-400">·</span>
          <span className="text-base text-gray-300">{customerName(order, UNKNOWN_CUSTOMER_LABEL)}</span>
        </div>
        {shippingAddressLine(order.shipping) && (
          <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
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
            {nameFor(heldByOther)} is packing this order{detail?.session.locked_since ? ` · ${formatLockAge(detail.session.locked_since)}` : ''} -- view only.
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
          const packedQty = packedByLineItem.get(li.id) ?? 0
          const done = packedQty >= li.quantity
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setPackLine({ lineItem: li, packedSoFar: packedQty })}
              disabled={readOnly}
              className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-accent transition-colors"
            >
              <ProductThumb token={token} productId={li.product_id} appearance={SCANNER_THUMB} title={li.name} />
              {done && <Check size={16} className="text-green-400 shrink-0" />}
              <div className="flex-1 min-w-0">
                <p className="text-sm truncate">{li.name}</p>
                {li.sku_code && <p className="text-xs text-gray-400 truncate">{li.sku_code}</p>}
              </div>
              <p className={`text-sm font-medium shrink-0 ${done ? 'text-green-400' : 'text-gray-300'}`}>
                {packedQty} / {li.quantity}
              </p>
            </button>
          )
        })}

        {extras.length > 0 && (
          <ul aria-label="Extra items" className="space-y-2">
            {extras.map(s => (
              <li key={s.id} className="flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3">
                <Gift size={16} className="text-gray-400 shrink-0" />
                <p className="flex-1 min-w-0 text-sm truncate">{packScanLabel(s)}</p>
                <button aria-label={`Remove ${s.reason || 'extra item'}`}
                  type="button" onClick={() => handleDeleteScan(s.id)} title="Remove"
                  className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                >
                  <X size={14} />
                </button>
              </li>
            ))}
          </ul>
        )}

        <button
          type="button"
          onClick={() => setAddItemOpen(true)} disabled={readOnly}
          className="w-full flex items-center justify-center gap-1.5 rounded-xl border border-dashed border-night-700 px-3.5 py-3 text-sm text-accent hover:border-accent transition-colors"
        >
          <Gift size={14} /> Add freebie or new item
        </button>
      </div>

      <form onSubmit={handleScan} className="border-t border-night-700 p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
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

      {packLine && (
        <LineChangeSheet
          token={token}
          lineItem={packLine.lineItem}
          packedSoFar={packLine.packedSoFar}
          scans={(detail?.scans ?? []).filter(s => s.line_item_id === packLine.lineItem.id)}
          onDeleteScan={handleDeleteScan}
          onCancel={() => setPackLine(null)}
          onPack={quantity => {
            setPackLine(null)
            stage({ sku_id: packLine.lineItem.sku_id ?? undefined, quantity })
          }}
          onSubstitute={lines => {
            setPackLine(null)
            run(() => api.substitutePackLine(token, sessionId, { line_item_id: packLine.lineItem.id, lines }), 'Substitute failed')
          }}
          onRefund={(quantity, reason) => {
            setPackLine(null)
            run(() => api.refundPackLine(token, sessionId, { line_item_id: packLine.lineItem.id, quantity, reason }), 'Refund failed')
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

      {labelModals}

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
              This discards every item packed so far on this order and starts the session fresh. This can't be undone.
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
