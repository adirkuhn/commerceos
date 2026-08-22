import { useState, useEffect, useRef, useCallback } from 'react'
import { ScanBarcode, CheckCircle, Package, Search, PauseCircle, Repeat, Gift, PackagePlus, X, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as omsApi from '../oms/api'
import type { SalesOrder } from '../oms/types'
import * as pickingApi from '../scanner/api'
import type { PickingSessionDetail, PickingSessionLineItem, PickScan, CompletePickingResult } from '../scanner/types'

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

// A pick or substitute scan's net effect on a line's picked progress --
// mirrors the backend's own missingPickItems scoring exactly (freebie/
// new_item never count against an original line, they have no line_item_id).
function pickedQtyByLineItem(scans: PickScan[]): Map<string, number> {
  const byLineItem = new Map<string, number>()
  for (const s of scans) {
    if (!s.line_item_id) continue
    if (s.action_type !== 'pick' && s.action_type !== 'substitute') continue
    byLineItem.set(s.line_item_id, (byLineItem.get(s.line_item_id) ?? 0) + s.quantity)
  }
  return byLineItem
}

const ACTION_LABEL: Record<PickScan['action_type'], string> = {
  pick: 'Picked', substitute: 'Substituted', freebie: 'Freebie', new_item: 'New item',
}

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
  const codeRef = useRef<HTMLInputElement>(null)

  const [pickLine, setPickLine] = useState<{ lineItem: PickingSessionLineItem; pickedSoFar: number } | null>(null)
  const [pickQty, setPickQty] = useState(1)
  const [substituteMode, setSubstituteMode] = useState(false)
  const [substituteCode, setSubstituteCode] = useState('')

  const [addItemOpen, setAddItemOpen] = useState<{ skuId: string; productName: string } | null>(null)
  const [addItemQty, setAddItemQty] = useState(1)
  const [addItemPrice, setAddItemPrice] = useState('0')
  const [addItemFreebie, setAddItemFreebie] = useState(true)

  const [pausing, setPausing] = useState(false)
  const [completing, setCompleting] = useState(false)
  const [completeResult, setCompleteResult] = useState<CompletePickingResult | null>(null)

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

  async function selectOrder(order: SalesOrder) {
    setError('')
    setCompleteResult(null)
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
      await loadSession(newSession.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start picking')
      setSessionLoading(false)
    }
  }

  function backToOrders() {
    setActiveOrder(null)
    setSession(null)
    setCompleteResult(null)
    setError('')
  }

  async function handleScanSubmit() {
    const scannedCode = code.trim()
    if (!scannedCode || !session || !token) return
    setError('')
    setBusy(true)
    try {
      const resolved = await pickingApi.resolvePickScanCode(token, session.session.id, scannedCode)
      setCode('')
      const li = session.line_items.find(l => l.sku_id === resolved.sku_id)
      if (li) {
        const pickedSoFar = pickedQtyByLineItem(session.scans).get(li.id) ?? 0
        setPickLine({ lineItem: li, pickedSoFar })
        setPickQty(Math.max(1, li.quantity - pickedSoFar))
        setSubstituteMode(false)
      } else {
        setAddItemOpen({ skuId: resolved.sku_id, productName: resolved.product_name })
        setAddItemQty(1)
        setAddItemPrice('0')
        setAddItemFreebie(true)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Scan not recognized')
    } finally {
      setBusy(false)
      codeRef.current?.focus()
    }
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
    setSubstituteMode(false)
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

  async function confirmSubstitute() {
    if (!pickLine || !session || !token) return
    const scannedCode = substituteCode.trim()
    if (!scannedCode) { setError('Scan or enter the substitute SKU/barcode'); return }
    setBusy(true)
    setError('')
    try {
      const resolved = await pickingApi.resolvePickScanCode(token, session.session.id, scannedCode)
      await pickingApi.scanPickItem(token, session.session.id, {
        sku_id: resolved.sku_id, line_item_id: pickLine.lineItem.id, action_type: 'substitute',
      })
      setPickLine(null)
      setSubstituteMode(false)
      setSubstituteCode('')
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record substitute')
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

  async function handleComplete() {
    if (!session || !token) return
    setCompleting(true)
    setError('')
    try {
      setCompleteResult(await pickingApi.completePicking(token, session.session.id))
      await loadOrders()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete picking')
    } finally {
      setCompleting(false)
    }
  }

  const pickedByLineItem = session ? pickedQtyByLineItem(session.scans) : new Map<string, number>()
  const totalItems = session?.line_items.reduce((s, li) => s + li.quantity, 0) ?? 0
  const pickedItems = session ? session.line_items.reduce((s, li) => s + Math.min(li.quantity, pickedByLineItem.get(li.id) ?? 0), 0) : 0
  const progress = totalItems > 0 ? (pickedItems / totalItems) * 100 : 0
  const allPicked = totalItems > 0 && pickedItems >= totalItems

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
          <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">Orders to Pick</p>
          <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{orders.length} unfulfilled</p>
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
            <p className="px-4 py-3 text-[11px] dark:text-gray-500 text-gray-400">No unfulfilled orders</p>
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

      {/* Main pick area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!activeOrder && (
          <div className="flex-1 flex flex-col items-center justify-center dark:text-gray-500 text-gray-400">
            <ScanBarcode size={32} className="mb-3 opacity-40" />
            <p className="text-sm">Select an order to start picking</p>
          </div>
        )}

        {activeOrder && sessionLoading && !session && (
          <div className="flex-1 flex items-center justify-center dark:text-gray-500 text-gray-400 text-sm">Loading…</div>
        )}

        {activeOrder && completeResult && (
          <div className="flex-1 flex flex-col items-center justify-center p-8">
            <CheckCircle size={40} className="text-green-400 mb-3" />
            <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">
              #{activeOrder.number || activeOrder.external_order_id} marked fulfilled
            </p>
            {completeResult.missing_items.length > 0 && (
              <div className="mt-4 w-full max-w-sm rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-3">
                <p className="text-[11px] font-medium text-amber-400 mb-2">Not fully picked</p>
                {completeResult.missing_items.map(m => (
                  <p key={m.line_item_id} className="text-[11px] dark:text-gray-300 text-gray-600">
                    {m.name}: {m.picked_qty}/{m.expected_qty}
                  </p>
                ))}
              </div>
            )}
            <button
              type="button" onClick={backToOrders}
              className="mt-5 px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700"
            >
              Back to orders
            </button>
          </div>
        )}

        {activeOrder && session && !completeResult && (
          <>
            {/* Header */}
            <div className={clsx(
              'flex items-center gap-4 px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0',
            )}>
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-0.5">
                  <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">#{activeOrder.number || activeOrder.external_order_id}</h2>
                  <span className="text-xs dark:text-gray-500 text-gray-400">·</span>
                  <span className="text-xs dark:text-gray-400 text-gray-600">{customerName(activeOrder)}</span>
                </div>
                <p className="text-xs dark:text-gray-500 text-gray-400">{formatMoney(activeOrder.total_minor_units, activeOrder.currency_code)}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button" onClick={handlePause} disabled={pausing}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-medium dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50"
                >
                  <PauseCircle size={13} /> {pausing ? 'Pausing…' : 'Pause'}
                </button>
                <button
                  type="button" onClick={() => setAddItemOpen({ skuId: '', productName: '' })}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-medium dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50"
                >
                  <PackagePlus size={13} /> Add item
                </button>
                <button
                  type="button" onClick={handleComplete} disabled={completing}
                  className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700 disabled:opacity-50"
                >
                  <CheckCircle size={13} /> {completing ? 'Completing…' : 'Complete'}
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
                  disabled={busy || !!pickLine || !!addItemOpen}
                  className="flex-1 bg-transparent text-sm dark:text-gray-100 text-gray-900 outline-none dark:placeholder-gray-500 placeholder-gray-400 font-mono disabled:opacity-50"
                />
              </div>
              {error && <p className="text-[11px] text-red-400 mt-2">{error}</p>}
            </div>

            {/* Progress */}
            <div className="px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs dark:text-gray-400 text-gray-500">Progress</span>
                <span className="text-xs font-medium dark:text-gray-200 text-gray-800">{pickedItems} / {totalItems} items</span>
              </div>
              <div className="h-2 dark:bg-[#2D3748] bg-gray-200 rounded-full overflow-hidden">
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
                      'dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200',
                    )}
                  >
                    <div className={clsx('w-10 h-10 rounded-lg flex items-center justify-center shrink-0', done ? 'bg-green-500/20' : 'dark:bg-[#2D3748] bg-gray-100')}>
                      {done ? <CheckCircle size={18} className="text-green-400" /> : <Package size={18} className="dark:text-gray-500 text-gray-400" />}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className={clsx('text-sm font-medium', done ? 'dark:text-green-300 text-green-700' : 'dark:text-gray-100 text-gray-900')}>{li.name}</p>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{li.sku_code}</span>
                        {!li.sku_id && <span className="text-[10px] text-amber-400">unlinked — scan to resolve</span>}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <p className={clsx('text-lg font-bold tabular-nums', done ? 'text-green-400' : partial ? 'text-amber-400' : 'dark:text-gray-200 text-gray-800')}>
                        {picked}/{li.quantity}
                      </p>
                      <p className="text-[10px] dark:text-gray-500 text-gray-400">picked</p>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Staged scans log */}
            {session.scans.length > 0 && (
              <div className="border-t dark:border-[#2D3748] border-gray-200 px-4 py-2 max-h-32 overflow-y-auto shrink-0">
                <p className="text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1">This session</p>
                {session.scans.map(s => (
                  <div key={s.id} className="flex items-center justify-between py-0.5">
                    <span className="text-[11px] dark:text-gray-400 text-gray-600">
                      {ACTION_LABEL[s.action_type]} · qty {s.quantity}
                    </span>
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

      {/* Pick / Substitute confirm panel */}
      {pickLine && (
        <div className="w-80 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col shrink-0">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">{pickLine.lineItem.name}</p>
            <button type="button" onClick={() => setPickLine(null)} className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300">
              <X size={14} />
            </button>
          </div>
          <div className="p-4 space-y-4">
            <div className="flex items-center rounded-lg border dark:border-[#2D3748] border-gray-200 overflow-hidden">
              <button
                type="button" onClick={() => setSubstituteMode(false)}
                className={clsx('flex-1 px-3 py-1.5 text-[11px] font-medium', !substituteMode ? 'bg-blue-600 text-white' : 'dark:bg-[#1F2937] bg-white dark:text-gray-400 text-gray-500')}
              >
                Pick
              </button>
              <button
                type="button" onClick={() => setSubstituteMode(true)}
                className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-[11px] font-medium', substituteMode ? 'bg-blue-600 text-white' : 'dark:bg-[#1F2937] bg-white dark:text-gray-400 text-gray-500')}
              >
                <Repeat size={11} /> Substitute
              </button>
            </div>

            {!substituteMode ? (
              <>
                <div>
                  <label className="block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">Quantity</label>
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
                <p className="text-[11px] dark:text-gray-400 text-gray-500">
                  Scan the replacement product. The full remaining quantity ({pickLine.lineItem.quantity - pickLine.pickedSoFar}) will be substituted.
                </p>
                <input
                  type="text" value={substituteCode} onChange={e => setSubstituteCode(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && confirmSubstitute()}
                  placeholder="Scan or type replacement SKU/barcode"
                  className={inputClass} autoFocus
                />
                <button
                  type="button" onClick={confirmSubstitute} disabled={busy}
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
        <div className="w-80 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col shrink-0">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">Add item not on order</p>
            <button type="button" onClick={() => setAddItemOpen(null)} className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300">
              <X size={14} />
            </button>
          </div>
          <div className="p-4 space-y-4">
            {!addItemOpen.skuId ? (
              <>
                <p className="text-[11px] dark:text-gray-400 text-gray-500">Scan or type the product's SKU/barcode.</p>
                <input
                  type="text" value={code} onChange={e => setCode(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleAddItemScanSubmit()}
                  placeholder="Scan or type SKU/barcode" className={inputClass} autoFocus
                />
              </>
            ) : (
              <>
                <p className="text-xs dark:text-gray-200 text-gray-800">{addItemOpen.productName}</p>
                <div className="flex items-center rounded-lg border dark:border-[#2D3748] border-gray-200 overflow-hidden">
                  <button
                    type="button" onClick={() => setAddItemFreebie(true)}
                    className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-[11px] font-medium', addItemFreebie ? 'bg-blue-600 text-white' : 'dark:bg-[#1F2937] bg-white dark:text-gray-400 text-gray-500')}
                  >
                    <Gift size={11} /> Freebie
                  </button>
                  <button
                    type="button" onClick={() => setAddItemFreebie(false)}
                    className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-[11px] font-medium', !addItemFreebie ? 'bg-blue-600 text-white' : 'dark:bg-[#1F2937] bg-white dark:text-gray-400 text-gray-500')}
                  >
                    <PackagePlus size={11} /> New item
                  </button>
                </div>
                <div>
                  <label className="block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">Quantity</label>
                  <input
                    type="number" min={1} value={addItemQty}
                    onChange={e => setAddItemQty(Math.max(1, parseInt(e.target.value, 10) || 1))}
                    className={inputClass}
                  />
                </div>
                {!addItemFreebie && (
                  <div>
                    <label className="block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">Price</label>
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

    </div>
  )
}
