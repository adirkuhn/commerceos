import { useState, useEffect, useRef, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ScanBarcode, CheckCircle, Package, Search, X, Trash2, ArrowLeft, Undo2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as omsApi from '../oms/api'
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

// Duplicated from Picking.tsx on purpose -- same small-helper convention
// used elsewhere in this codebase rather than a shared cross-screen import.
function LineItemThumb({ token, productId }: { token: string; productId?: string | null }) {
  const { media } = useProductMedia(token, productId ?? null)
  const featured = media[0]
  return (
    <div className="w-10 h-10 rounded-lg dark:bg-[#2D3748] bg-gray-100 flex items-center justify-center shrink-0 overflow-hidden">
      {featured ? (
        <img src={featured.url} alt={featured.alt_text} className="w-full h-full object-cover" />
      ) : (
        <Package size={18} className="dark:text-gray-500 text-gray-400" />
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

export default function Packing() {
  const { token } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()

  const [orders, setOrders] = useState<SalesOrder[]>([])
  const [ordersLoading, setOrdersLoading] = useState(true)
  const [ordersError, setOrdersError] = useState('')
  const [search, setSearch] = useState('')

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

  const loadOrders = useCallback(async () => {
    if (!token) return
    setOrdersLoading(true)
    setOrdersError('')
    try {
      const result = await omsApi.listSalesOrders(token, { fulfilled: true, packed: false, perPage: 100 })
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

  async function selectOrder(order: SalesOrder) {
    setError('')
    setCompleteResult(null)
    setPackLine(null)
    setActiveOrder(order)
    setSession(null)
    setSessionLoading(true)
    try {
      const newSession = await packingApi.startOrResumePacking(token ?? '', order.id)
      await loadSession(newSession.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start packing')
      setSessionLoading(false)
    }
  }

  // Deep-link entry point: Picking.tsx's "Continue to Packing" button lands
  // here with ?order=<id> so a picker who just finished can go straight
  // into packing the same order in one click, without hunting it in the
  // list -- still the same shared order list/lock underneath, so a
  // different person picking it up later from the list works identically.
  useEffect(() => {
    const orderId = searchParams.get('order')
    if (!orderId || !token || ordersLoading || activeOrder) return
    const order = orders.find(o => o.id === orderId)
    if (order) {
      selectOrder(order)
      setSearchParams(params => { params.delete('order'); return params }, { replace: true })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, token, ordersLoading, orders, activeOrder])

  function backToOrders() {
    setActiveOrder(null)
    setSession(null)
    setCompleteResult(null)
    setError('')
    setCreatedShipmentId(null)
    setLabelStep(null)
    setPendingRecipient(null)
    setConfirmingFinish(false)
    setExistingShipment(null)
  }

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
      await loadOrders()
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

  const filteredOrders = orders.filter(o => {
    if (!search.trim()) return true
    const q = search.toLowerCase()
    return (o.number || o.external_order_id).toLowerCase().includes(q) || customerName(o).toLowerCase().includes(q)
  })

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500'

  return (
    <div className="flex h-full">
      {/* Order queue */}
      <div className="w-64 border-r dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col overflow-hidden shrink-0">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">Orders to Pack</p>
          <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{orders.length} ready to pack</p>
        </div>
        <div className="px-3 py-2 border-b dark:border-[#2D3748] border-gray-200">
          <div className="relative">
            <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search orders..."
              className="w-full pl-7 pr-2 py-1.5 text-[11px] rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {ordersLoading && <p className="px-4 py-3 text-[11px] dark:text-gray-500 text-gray-400">Loading…</p>}
          {ordersError && <p className="px-4 py-3 text-[11px] text-red-400">{ordersError}</p>}
          {!ordersLoading && filteredOrders.length === 0 && (
            <p className="px-4 py-3 text-[11px] dark:text-gray-500 text-gray-400">No orders ready to pack</p>
          )}
          {filteredOrders.map(o => (
            <button
              key={o.id}
              onClick={() => selectOrder(o)}
              className={clsx(
                'w-full text-left px-4 py-3 border-b dark:border-[#2D3748] border-gray-100 transition-colors',
                activeOrder?.id === o.id
                  ? 'dark:bg-blue-600/10 bg-blue-50 border-l-2 border-l-blue-500'
                  : 'hover:dark:bg-[#1F2937] hover:bg-white',
              )}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs font-medium dark:text-gray-200 text-gray-800">#{o.number || o.external_order_id}</span>
                {activeOrder?.id === o.id && session?.session.status === 'in_progress' && (
                  <span className="text-[9px] font-medium text-blue-400">In progress</span>
                )}
              </div>
              <p className="text-[10px] dark:text-gray-400 text-gray-500 truncate">{customerName(o)}</p>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{formatMoney(o.total_minor_units, o.currency_code)}</p>
            </button>
          ))}
        </div>
      </div>

      {/* Main pack area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!activeOrder && (
          <div className="flex-1 flex flex-col items-center justify-center dark:text-gray-500 text-gray-400">
            <Package size={32} className="mb-3 opacity-40" />
            <p className="text-sm">Select an order to start packing</p>
          </div>
        )}

        {activeOrder && sessionLoading && !session && (
          <div className="flex-1 flex items-center justify-center dark:text-gray-500 text-gray-400 text-sm">Loading…</div>
        )}

        {activeOrder && completeResult && (
          <div className="flex-1 flex flex-col items-center justify-center p-8">
            <CheckCircle size={40} className="text-green-400 mb-3" />
            <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">
              #{activeOrder.number || activeOrder.external_order_id} packed
            </p>
            {!!completeResult.missing_items?.length && (
              <div className="mt-4 w-full max-w-sm rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-3">
                <p className="text-[11px] font-medium text-amber-400 mb-2">Not fully packed</p>
                {completeResult.missing_items.map(m => (
                  <p key={m.line_item_id} className="text-[11px] dark:text-gray-300 text-gray-600">
                    {m.name}: {m.packed_qty}/{m.expected_qty}
                  </p>
                ))}
              </div>
            )}

            {myParcelConnection && !createdShipmentId && (
              <button
                type="button" onClick={handleCreateLabelClick}
                className="mt-5 px-4 py-2 rounded-lg dark:bg-[#1F2937] bg-gray-100 dark:text-gray-200 text-gray-700 text-xs font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
              >
                Create shipping label
              </button>
            )}
            {createdShipmentId && (
              <button
                type="button" onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
                className="mt-5 px-4 py-2 rounded-lg dark:bg-[#1F2937] bg-gray-100 dark:text-gray-200 text-gray-700 text-xs font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
              >
                {autoPrintEnabled ? 'Retry print' : 'Print shipping label'}
              </button>
            )}

            <button
              type="button" onClick={backToOrders}
              className="mt-3 px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700"
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
            <div className="flex items-center gap-3 px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <button onClick={() => setConfirmingFinish(false)} className="p-1 -ml-1" title="Back to packing">
                <ArrowLeft size={18} className="dark:text-gray-400 text-gray-500" />
              </button>
              <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 flex-1">Review before completing</h2>
            </div>
            <div className="px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <p className="text-sm font-medium dark:text-gray-200 text-gray-800">
                #{activeOrder.number || activeOrder.external_order_id} · {customerName(activeOrder)}
              </p>
              {shippingAddressLine(activeOrder.shipping) && (
                <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">{shippingAddressLine(activeOrder.shipping)}</p>
              )}
            </div>
            <div className="flex-1 overflow-y-auto px-6 py-4 space-y-2">
              {(() => {
                const missing = computeMissingPackItems(session.line_items, session.scans)
                return missing.length === 0 ? (
                  <p className="text-sm dark:text-gray-400 text-gray-500">Every line is fully packed.</p>
                ) : (
                  <>
                    <p className="text-sm text-amber-400">{missing.length} line(s) aren't fully packed yet:</p>
                    {missing.map(m => (
                      <div key={m.line_item_id} className="rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 px-3 py-2">
                        <p className="text-sm dark:text-gray-200 text-gray-800">{m.name}</p>
                        <p className="text-xs dark:text-gray-500 text-gray-400">expected {m.expected_qty} · packed {m.packed_qty}</p>
                      </div>
                    ))}
                  </>
                )
              })()}
            </div>
            <div className="p-6 border-t dark:border-[#2D3748] border-gray-200 shrink-0 space-y-2">
              {error && <p className="text-sm text-red-400">{error}</p>}
              <button
                onClick={handleConfirmComplete} disabled={completing}
                className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
              >
                {completing ? 'Completing…' : 'Confirm & Complete'}
              </button>
              <button
                onClick={() => setConfirmingFinish(false)}
                className="w-full py-2.5 rounded-xl dark:bg-[#1F2937] bg-gray-100 dark:text-gray-300 text-gray-700 text-sm font-medium"
              >
                Back to packing
              </button>
            </div>
          </div>
        )}

        {activeOrder && session && !completeResult && !confirmingFinish && (
          <>
            {/* Header */}
            <div className="flex items-center gap-4 px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-0.5">
                  <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">#{activeOrder.number || activeOrder.external_order_id}</h2>
                  <span className="text-xs dark:text-gray-500 text-gray-400">·</span>
                  <span className="text-xs dark:text-gray-400 text-gray-600">{customerName(activeOrder)}</span>
                </div>
                <p className="text-xs dark:text-gray-500 text-gray-400">{formatMoney(activeOrder.total_minor_units, activeOrder.currency_code)}</p>
                {shippingAddressLine(activeOrder.shipping) && (
                  <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">{shippingAddressLine(activeOrder.shipping)}</p>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button" onClick={() => setResetConfirmOpen(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-medium dark:text-red-400 text-red-500 hover:dark:bg-[#2D3748] hover:bg-gray-50"
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
            <div className="px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 shrink-0 dark:bg-[#111827]/80 bg-gray-50">
              <div className="flex-1 flex items-center gap-3 px-4 py-3 rounded-xl border-2 dark:border-blue-500 border-blue-400 dark:bg-[#1F2937] bg-white">
                <ScanBarcode size={20} className="text-blue-400" />
                <input
                  ref={codeRef}
                  type="text"
                  value={code}
                  onChange={e => setCode(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleScanSubmit()}
                  placeholder="Scan barcode or type and press Enter..."
                  disabled={busy || !!packLine || !!scanErrorAlert}
                  className="flex-1 bg-transparent text-sm dark:text-gray-100 text-gray-900 outline-none dark:placeholder-gray-500 placeholder-gray-400 font-mono disabled:opacity-50"
                />
              </div>
              {error && <p className="text-[11px] text-red-400 mt-2">{error}</p>}
            </div>

            {/* Progress */}
            <div className="px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs dark:text-gray-400 text-gray-500">Progress</span>
                <span className="text-xs font-medium dark:text-gray-200 text-gray-800">{packedItems} / {totalItems} items</span>
              </div>
              <div className="h-2 dark:bg-[#2D3748] bg-gray-200 rounded-full overflow-hidden">
                <div
                  className={clsx('h-full rounded-full transition-all duration-500', allPacked ? 'bg-green-500' : 'bg-blue-500')}
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
                      'dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200',
                    )}
                  >
                    {done ? (
                      <div className="w-10 h-10 rounded-lg bg-green-500/20 flex items-center justify-center shrink-0">
                        <CheckCircle size={18} className="text-green-400" />
                      </div>
                    ) : (
                      <LineItemThumb token={token ?? ''} productId={li.product_id} />
                    )}
                    <div className="flex-1 min-w-0">
                      <p className={clsx('text-sm font-medium', done ? 'dark:text-green-300 text-green-700' : 'dark:text-gray-100 text-gray-900')}>{li.name}</p>
                      <span className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{li.sku_code}</span>
                    </div>
                    <div className="text-right shrink-0">
                      <p className={clsx('text-lg font-bold tabular-nums', done ? 'text-green-400' : partial ? 'text-amber-400' : 'dark:text-gray-200 text-gray-800')}>
                        {packed}/{li.quantity}
                      </p>
                      <p className="text-[10px] dark:text-gray-500 text-gray-400">packed</p>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Staged scans log */}
            {session.scans.length > 0 && (
              <div className="border-t dark:border-[#2D3748] border-gray-200 px-4 py-2 max-h-32 overflow-y-auto shrink-0">
                <p className="text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1">This session</p>
                {session.scans.map((s: PackScan) => (
                  <div key={s.id} className="flex items-center justify-between py-0.5">
                    <span className="text-[11px] dark:text-gray-400 text-gray-600">Packed · qty {s.quantity}</span>
                    <button type="button" onClick={() => handleDeleteScan(s.id)} aria-label="Remove scan" className="dark:text-gray-600 text-gray-400 hover:text-red-400">
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
        <div className="w-80 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col shrink-0">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">{packLine.lineItem.name}</p>
            <button type="button" onClick={() => setPackLine(null)} className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300">
              <X size={14} />
            </button>
          </div>
          <div className="p-4 space-y-4">
            <div>
              <label className="block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">Quantity</label>
              <input
                type="number" min={1} max={packLine.lineItem.quantity - packLine.packedSoFar || undefined}
                value={packQty} onChange={e => setPackQty(Math.max(1, parseInt(e.target.value, 10) || 1))}
                className={inputClass}
              />
            </div>
            <button
              type="button" onClick={confirmPack} disabled={busy}
              className="w-full px-3 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50"
            >
              Confirm pack
            </button>
          </div>
        </div>
      )}

      {scanErrorAlert && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-xs rounded-xl dark:bg-[#111827] bg-white border dark:border-red-500/40 border-red-300 p-5 text-center shadow-xl">
            <p className="text-sm font-semibold text-red-400 mb-4">{scanErrorAlert}</p>
            <button
              type="button" onClick={dismissScanError}
              className="w-full py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700"
            >
              OK
            </button>
          </div>
        </div>
      )}

      {resetConfirmOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-xs rounded-xl dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 p-5 text-center shadow-xl">
            <p className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Start over?</p>
            <p className="text-xs dark:text-gray-400 text-gray-500 mb-4">
              This discards every item packed so far on this order and starts the session fresh. This can't be undone.
            </p>
            <div className="flex gap-2">
              <button
                type="button" onClick={() => setResetConfirmOpen(false)} disabled={resetting}
                className="flex-1 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 text-xs font-medium dark:text-gray-300 text-gray-700 disabled:opacity-50"
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
        </div>
      )}
    </div>
  )
}
