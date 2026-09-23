import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, Camera, X, Package, Undo2 } from 'lucide-react'
import * as api from './api'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { QuantityStepper } from './QuantityStepper'
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
import type { PackingSessionDetail, PickingSessionLineItem, PackScan, CompletePackingResult } from './types'
import type { SalesOrder, Address } from '../oms/types'
import { Dialog } from '../components/common/Dialog'

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

interface PackingScreenProps {
  token: string
  sessionId: string
  order: SalesOrder
  onExit: () => void
  onAuthError: () => void
}

function isAuthError(err: unknown): boolean {
  return err instanceof Error && /401|unauthorized/i.test(err.message)
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

  // Heartbeat -- see PickingScreen's identical effect for the rationale.
  useEffect(() => {
    if (detail?.session.status === 'completed') return
    const interval = setInterval(() => {
      api.touchPackingSession(token, sessionId).catch(err => {
        if (isAuthError(err)) return onAuthError()
        if (err instanceof Error && /locked/i.test(err.message)) {
          setError('Another user has taken over this packing session')
        }
      })
    }, 60_000)
    return () => clearInterval(interval)
  }, [sessionId, detail?.session.status, token, onAuthError])
  useWakeLock()

  async function stage(input: Parameters<typeof api.scanPackItem>[2]): Promise<boolean> {
    setError('')
    setScanning(true)
    try {
      await api.scanPackItem(token, sessionId, input)
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

  async function handleConfirmComplete() {
    setFinishing(true)
    try {
      const r = await api.completePacking(token, sessionId)
      setResult(r)
      setConfirmingFinish(false)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to complete')
    } finally {
      setFinishing(false)
    }
  }

  if (confirmingFinish && detail) {
    const missing = computeMissingPackItems(detail.line_items, detail.scans)
    return (
      <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
          <button onClick={() => setConfirmingFinish(false)} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to packing">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
        </div>
        <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
          <div className="flex items-baseline gap-1.5 flex-wrap">
            <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
            <span className="text-gray-400">·</span>
            <span className="text-base text-gray-300">{customerName(order)}</span>
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
          {myParcelConnection && !createdShipmentId && (
            <button
              type="button" onClick={handleCreateLabelClick}
              className="w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-200 text-sm font-medium hover:border-blue-500 transition-colors"
            >
              Create shipping label
            </button>
          )}
          {createdShipmentId && (
            <button
              type="button" onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
              className="w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-200 text-sm font-medium hover:border-blue-500 transition-colors"
            >
              {autoPrintEnabled ? 'Retry print' : 'Print shipping label'}
            </button>
          )}
          <button onClick={onExit} className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700">
            Done
          </button>
        </div>

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
      </div>
    )
  }

  const packedByLineItem = packedQuantityByLineItem(detail?.scans ?? [])

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button onClick={onExit} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Packing</h1>
        <button
          onClick={() => setConfirmingFinish(true)}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700"
        >
          Finish
        </button>
      </div>

      <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
        <div className="flex items-baseline gap-1.5 flex-wrap">
          <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
          <span className="text-gray-400">·</span>
          <span className="text-base text-gray-300">{customerName(order)}</span>
        </div>
        {shippingAddressLine(order.shipping) && (
          <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
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
          const packedQty = packedByLineItem.get(li.id) ?? 0
          const done = packedQty >= li.quantity
          return (
            <button
              key={li.id}
              type="button"
              onClick={() => setPackLine({ lineItem: li, packedSoFar: packedQty })}
              className="w-full flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3 text-left hover:border-blue-500 transition-colors"
            >
              <LineItemThumb token={token} name={li.name} productId={li.product_id} />
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
      </div>

      <form onSubmit={handleScan} className="border-t border-night-700 p-4 space-y-2 shrink-0">
        {error && <p className="text-sm text-red-400">{error}</p>}
        <div className="flex gap-2">
          <input
            ref={codeInputRef}
            value={code}
            onChange={e => setCode(e.target.value)}
            placeholder="Scan or type barcode…"
            disabled={scanning || !!scanErrorAlert}
            className="flex-1 px-3 py-3 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
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
          className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50"
        >
          {scanning ? 'Looking up…' : 'Scan'}
        </button>
      </form>

      {packLine && (
        <PackLineSheet
          lineItem={packLine.lineItem}
          packedSoFar={packLine.packedSoFar}
          scans={(detail?.scans ?? []).filter(s => s.line_item_id === packLine.lineItem.id)}
          onDeleteScan={handleDeleteScan}
          onCancel={() => setPackLine(null)}
          onPack={quantity => {
            setPackLine(null)
            stage({ sku_id: packLine.lineItem.sku_id ?? undefined, quantity })
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

function PackLineSheet({ lineItem, packedSoFar, scans, onDeleteScan, onCancel, onPack }: {
  lineItem: PickingSessionLineItem
  packedSoFar: number
  scans: PackScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onPack: (quantity: number) => void
}) {
  const remaining = Math.max(lineItem.quantity - packedSoFar, 0)
  const [qty, setQty] = useState(String(remaining))
  const n = Number(qty) || 0

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{lineItem.name}</h2>
            <p className="text-xs text-gray-400 mt-0.5">Packed {packedSoFar} of {lineItem.quantity}</p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        {scans.length > 0 && (
          <div className="px-5 pt-3 space-y-1.5">
            <p className="text-xs text-gray-400">Already packed for this line:</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center gap-2 rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <p className="flex-1 text-sm text-gray-200">Packed × {s.quantity}</p>
                <button
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title="Delete this scan"
                  className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                >
                  <X size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="p-5 space-y-4">
          <QuantityStepper id="pack-quantity" label="Quantity to pack" value={qty} onChange={setQty} min={0} max={remaining} />
          <button
            onClick={() => onPack(n)}
            disabled={n <= 0}
            className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold disabled:opacity-50"
          >
            Confirm pack
          </button>
        </div>
      </div>
    </Dialog>
  )
}
