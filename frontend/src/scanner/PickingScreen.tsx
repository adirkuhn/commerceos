import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, Check, PackageCheck, Camera, X, Package, Repeat, Gift, PackagePlus, Search } from 'lucide-react'
import * as api from './api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { useProductMedia } from '../pms/hooks/useProductMedia'
import { QuantityStepper } from './QuantityStepper'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import { playErrorSound } from './errorSound'
import { useWakeLock } from './useWakeLock'
import { computeMissingItems, pickedQuantityByLineItem } from './pickingMissingItems'
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
import type {
  PickingSessionDetail, PickingSessionLineItem, PickScan, CompletePickingResult,
  StockShortfall, StockShortfallResolution, PickScanActionType, SubstituteLine,
} from './types'
import { StockShortfallDialog } from './StockShortfallDialog'
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
  const [stockShortfalls, setStockShortfalls] = useState<StockShortfall[] | null>(null)
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])
  const [pickLine, setPickLine] = useState<{ lineItem: PickingSessionLineItem; pickedSoFar: number } | null>(null)
  const [addItemOpen, setAddItemOpen] = useState(false)
  const codeInputRef = useRef<HTMLInputElement>(null)

  const { connections } = useConnections(token)
  const myParcelConnection = connections.find(c => c.provider === 'myparcel')
  // Same 3-step label flow as the desktop picking screens (see Picking.tsx's
  // doc comment): 'address' -> 'preview' -> Print label actually creates the
  // shipment (handlePrintLabel).
  const [labelStep, setLabelStep] = useState<'address' | 'preview' | null>(null)
  const [pendingRecipient, setPendingRecipient] = useState<RecipientAddress | null>(null)
  // null until AddressConfirmModal's onNext runs once -- lets its own
  // defaultCarrier (the shipping-method guess below) take effect on first
  // open instead of always being masked by a hardcoded initial value.
  const [pendingCarrier, setPendingCarrier] = useState<number | null>(null)
  const [labelSize, setLabelSize] = useState<'A4' | 'A6'>('A4')
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null)
  // See Picking.tsx's identical labelTab doc.
  const [labelTab, setLabelTab] = useState<Window | null>(null)
  // See Picking.tsx's identical existingShipment doc.
  const [existingShipment, setExistingShipment] = useState<Shipment | null>(null)
  const [autoPrintEnabled, setAutoPrintEnabled] = useState(false)
  const [connectionDefaultCarrier, setConnectionDefaultCarrier] = useState(1)
  useEffect(() => {
    if (!myParcelConnection) return
    shippingApi.getShippingSettings(token, myParcelConnection.id)
      .then(result => {
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
        setConnectionDefaultCarrier(result.settings?.default_carrier ?? 1)
      })
      .catch(() => setAutoPrintEnabled(false))
  }, [token, myParcelConnection])

  // See Picking.tsx's identical handleCreateLabelClick doc.
  async function handleCreateLabelClick() {
    if (!myParcelConnection) return
    const shipments = await shippingApi.listShipmentsForOrder(token, order.id)
    // A cancelled shipment (wrong carrier, "Delete & create new") no longer
    // blocks creating a fresh one -- same rule CreateShipmentUseCase's own
    // duplicate-guard applies server-side.
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  // See Picking.tsx's identical handleOpenLabel doc.
  async function handleOpenLabel(shipmentId: string) {
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  // See Picking.tsx's identical handlePrintLabel doc.
  async function handlePrintLabel() {
    if (!myParcelConnection || !pendingRecipient) return
    const tab = window.open('', '_blank')
    let shipment
    try {
      shipment = await shippingApi.createShipment(token, {
        carrier: pendingCarrier ?? connectionDefaultCarrier, recipient: pendingRecipient,
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
  async function routeResolvedSku(skuId: string) {
    const li = detail?.line_items.find(l => l.sku_id === skuId)
    if (li) {
      const pickedSoFar = pickedQuantityByLineItem(detail?.scans ?? []).get(li.id) ?? 0
      if (pickedSoFar >= li.quantity) {
        playErrorSound()
        setScanErrorAlert('Already fully picked')
        return
      }
      await stage({ line_item_id: li.id, sku_id: skuId, action_type: 'pick', quantity: 1 })
      return
    }
    // No line item already carries this sku_id -- still worth trying the
    // server: it may auto-link one of this order's still-unmatched Woo
    // lines (product name matches the scanned item) rather than this
    // always being a genuine "wrong order" scan (see
    // ScanPickItemUseCase.autoLinkUnmatchedLine).
    const ok = await stage({ sku_id: skuId, action_type: 'pick', quantity: 1 })
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
      <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
        <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
          <button onClick={() => setConfirmingFinish(false)} className="p-1 -ml-1" title="Back to picking">
            <ArrowLeft size={18} className="text-gray-400" />
          </button>
          <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
        </div>
        {/* Customer/address stays visible here too, not just during active
            picking -- staff confirming completion should never have to go
            back just to double-check who this ships to (or create the label
            below without knowing the destination). */}
        <div className="px-4 py-3 border-b border-[#2D3748] shrink-0 bg-[#111827]">
          <div className="flex items-baseline gap-1.5 flex-wrap">
            <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
            <span className="text-gray-500">·</span>
            <span className="text-base text-gray-300">{customerName(order)}</span>
          </div>
          {shippingAddressLine(order.shipping) && (
            <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
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
          {myParcelConnection && !createdShipmentId && (
            <button
              type="button" onClick={handleCreateLabelClick}
              className="w-full py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-200 text-sm font-medium hover:border-blue-500 transition-colors"
            >
              Create shipping label
            </button>
          )}
          {createdShipmentId && (
            <button
              type="button" onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
              className="w-full py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-200 text-sm font-medium hover:border-blue-500 transition-colors"
            >
              {autoPrintEnabled ? 'Retry print' : 'Print shipping label'}
            </button>
          )}
          <button
            onClick={() => handleConfirmComplete()}
            disabled={finishing}
            className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
          >
            {finishing ? 'Completing…' : 'Confirm & Complete'}
          </button>
          <button onClick={() => setConfirmingFinish(false)} className="w-full py-2.5 rounded-xl bg-[#1F2937] text-gray-300 text-sm font-medium">
            Back, scan more
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
            defaultCarrier={guessCarrierFromShippingMethod(order.shipping_method_title, order.shipping_method, connectionDefaultCarrier)}
            onNext={({ carrier, recipient }) => { setPendingCarrier(carrier); setPendingRecipient(recipient); setLabelStep('preview') }}
            onClose={() => setLabelStep(null)}
          />
        )}
        {labelStep === 'preview' && pendingRecipient && (
          <LabelPreviewModal
            recipient={pendingRecipient}
            carrier={pendingCarrier ?? connectionDefaultCarrier}
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

  if (result) {
    return (
      <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
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
              <div key={m.line_item_id} className="rounded-lg bg-[#1F2937] border border-[#2D3748] px-3 py-2">
                <p className="text-sm text-gray-200">{m.name}</p>
                <p className="text-xs text-gray-500">expected {m.expected_qty} · picked {m.picked_qty}</p>
              </div>
            ))}
          </div>
        )}
        <div className="p-6 shrink-0 space-y-2">
          {myParcelConnection && !createdShipmentId && (
            <button
              type="button" onClick={handleCreateLabelClick}
              className="w-full py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-200 text-sm font-medium hover:border-blue-500 transition-colors"
            >
              Create shipping label
            </button>
          )}
          {createdShipmentId && (
            <button
              type="button" onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
              className="w-full py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-200 text-sm font-medium hover:border-blue-500 transition-colors"
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
            defaultCarrier={guessCarrierFromShippingMethod(order.shipping_method_title, order.shipping_method, connectionDefaultCarrier)}
            onNext={({ carrier, recipient }) => { setPendingCarrier(carrier); setPendingRecipient(recipient); setLabelStep('preview') }}
            onClose={() => setLabelStep(null)}
          />
        )}
        {labelStep === 'preview' && pendingRecipient && (
          <LabelPreviewModal
            recipient={pendingRecipient}
            carrier={pendingCarrier ?? connectionDefaultCarrier}
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

  const pickedByLineItem = pickedQuantityByLineItem(detail?.scans ?? [])
  const addedItems = (detail?.scans ?? []).filter(s => s.action_type === 'freebie' || s.action_type === 'new_item')

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={handlePause} className="p-1 -ml-1" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Picking</h1>
        <button
          onClick={() => setConfirmingFinish(true)}
          className="px-3 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700"
        >
          Finish
        </button>
      </div>

      {/* Order info -- its own full-width band, separate from the control
          row above, so the address has room to wrap instead of getting
          truncated next to the back/Finish buttons. */}
      <div className="px-4 py-3 border-b border-[#2D3748] shrink-0 bg-[#111827]">
        <div className="flex items-baseline gap-1.5 flex-wrap">
          <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
          <span className="text-gray-500">·</span>
          <span className="text-base text-gray-300">{customerName(order)}</span>
        </div>
        {shippingAddressLine(order.shipping) && (
          <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
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
          onSubstitute={lines => {
            setPickLine(null)
            stageSubstitute({ line_item_id: pickLine.lineItem.id, lines })
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
// it was picked, or -- when it's out of stock -- swap it for one or more
// replacement products. A single original unit can be split across
// several substitute SKUs (e.g. 1 unit for 3 of one cheaper item + 4 of
// another) -- each row picks its own SKU, how many of it is being added,
// how much of the original line it covers, and whether that covered share
// goes back to stock. The rows' covered quantities must sum to the
// original line's own quantity exactly before submitting.
type PickSheetMode = 'pick' | 'substitute'

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

function PickLineSheet({ token, lineItem, pickedSoFar, scans, onDeleteScan, onCancel, onPick, onSubstitute }: {
  token: string
  lineItem: PickingSessionLineItem
  pickedSoFar: number
  scans: PickScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onPick: (quantity: number, reason?: string) => void
  onSubstitute: (lines: SubstituteLine[]) => void
}) {
  const [mode, setMode] = useState<PickSheetMode>('pick')
  const [qty, setQty] = useState(String(Math.max(lineItem.quantity - pickedSoFar, 0)))
  const [reason, setReason] = useState('')
  const [substituteRows, setSubstituteRows] = useState<SubstituteRowState[]>([newSubstituteRow()])
  const n = Number(qty) || 0
  const remaining = Math.max(lineItem.quantity - pickedSoFar, 0)

  const coveredTotal = substituteRows.reduce((sum, r) => sum + (Number(r.coveredQty) || 0), 0)
  const substituteReady = substituteRows.length > 0
    && substituteRows.every(r => r.selected && Number(r.substituteQty) > 0 && Number(r.coveredQty) > 0)
    && coveredTotal === lineItem.quantity

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
                    {s.action_type === 'substitute'
                      ? `Substituted × ${s.quantity} (covers ${s.original_quantity_covered ?? '?'})`
                      : `Picked × ${s.quantity}`}
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
                Out of stock? Replace it with one or more products -- each row's "covers" quantity must add up to
                this line's own quantity ({lineItem.quantity}) before you can confirm.
              </p>
              {substituteRows.map((row, i) => (
                <div key={row.key} className="rounded-xl bg-[#1F2937] border border-[#2D3748] p-3 space-y-2">
                  {row.selected ? (
                    <div className="flex items-center gap-2">
                      <div className="flex-1 min-w-0 text-sm text-gray-200 truncate">{row.selected.sku.sku_code}</div>
                      <button
                        type="button"
                        onClick={() => updateRow(row.key, { selected: undefined, candidates: [] })}
                        className="shrink-0 p-1 text-gray-500 hover:text-red-400"
                        title="Change product"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  ) : (
                    <>
                      <form onSubmit={e => handleRowSearch(row.key, e)} className="flex gap-2">
                        <div className="relative flex-1">
                          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
                          <input
                            value={row.query}
                            onChange={e => updateRow(row.key, { query: e.target.value })}
                            placeholder="Search or scan barcode…"
                            className="w-full pl-8 pr-3 py-2 text-sm rounded-xl bg-[#111827] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </div>
                        <button type="submit" disabled={row.searching || !row.query.trim()} className="px-4 rounded-xl bg-[#111827] border border-[#2D3748] text-sm text-gray-300 disabled:opacity-50">
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
                          <span className="block text-[11px] text-gray-500 mb-1">Qty added</span>
                          <input
                            type="number" min={1} value={row.substituteQty}
                            onChange={e => updateRow(row.key, { substituteQty: e.target.value })}
                            className="w-full px-3 py-2 text-sm rounded-xl bg-[#111827] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </label>
                        <label className="block">
                          <span className="block text-[11px] text-gray-500 mb-1">Covers (of original)</span>
                          <input
                            type="number" min={1} value={row.coveredQty}
                            onChange={e => updateRow(row.key, { coveredQty: e.target.value })}
                            className="w-full px-3 py-2 text-sm rounded-xl bg-[#111827] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </label>
                      </div>
                      <label className="flex items-center gap-2 text-xs text-gray-400">
                        <input
                          type="checkbox" checked={row.returnToStock}
                          onChange={e => updateRow(row.key, { returnToStock: e.target.checked })}
                          className="rounded border-[#2D3748] bg-[#111827]"
                        />
                        Return original quantity to stock
                      </label>
                    </>
                  )}
                  {substituteRows.length > 1 && (
                    <button
                      type="button"
                      onClick={() => setSubstituteRows(rows => rows.filter((_, idx) => idx !== i))}
                      className="text-xs text-gray-500 hover:text-red-400"
                    >
                      Remove this row
                    </button>
                  )}
                </div>
              ))}
              <button
                type="button"
                onClick={() => setSubstituteRows(rows => [...rows, newSubstituteRow()])}
                className="w-full py-2 rounded-xl border border-dashed border-[#2D3748] text-sm text-gray-400 hover:text-gray-200"
              >
                + Add another substitute
              </button>
              <p className="text-xs text-gray-500 text-center">
                Covers {coveredTotal} of {lineItem.quantity}
              </p>
              <button
                onClick={handleSubmitSubstitute}
                disabled={!substituteReady}
                className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold disabled:opacity-50"
              >
                Confirm substitute
              </button>
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
