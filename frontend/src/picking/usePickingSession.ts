import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSessionLock } from '../fulfillment/useSessionLock'
import { useRoutedSession } from '../oms/useRoutedSession'
import type { SalesOrder } from '../oms/types'
import * as pickingApi from '../scanner/api'
import type { PickingSessionDetail, CompletePickingResult, StockShortfall, StockShortfallResolution } from '../scanner/types'
import { PICKING_PATH } from './pickingDisplay'

interface PickingSessionOptions {
  token: string | null
  userId: string
  refreshQueue: () => Promise<void>
  // Called wherever the open order/session changes or restarts, so the
  // pick sheet never outlives the line it was opened for.
  onSelectionReset: () => void
}

// The lifecycle of the picking session on screen: which order is open (URL
// routed), its lock, and pausing / starting over / completing it.
export function usePickingSession({ token, userId, refreshQueue, onSelectionReset }: PickingSessionOptions) {
  const [activeOrder, setActiveOrder] = useState<SalesOrder | null>(null)
  const [session, setSession] = useState<PickingSessionDetail | null>(null)
  const [sessionLoading, setSessionLoading] = useState(false)
  const [error, setError] = useState('')

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

  // Opening an order only views it; the first scan takes the lock. While we
  // hold it we heartbeat it, and leaving the order releases it.
  const { heldByOther } = useSessionLock({
    sessionId: session?.session.id ?? null, me: userId,
    lockedBy: session?.session.locked_by, lockedAt: session?.session.locked_at,
    active: !!session && session.session.status !== 'completed',
    touch: id => pickingApi.touchPickingSession(token ?? '', id),
    release: id => pickingApi.releasePickingSessionLock(token ?? '', id),
    onLost: () => setError('Another user has taken over this picking session'),
  })
  const readOnly = heldByOther !== null

  async function handleTakeOver() {
    if (!session || !token) return
    setError('')
    try {
      await pickingApi.takeOverPickingSession(token, session.session.id)
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to take over')
    }
  }

  function clearSelection() {
    setActiveOrder(null)
    setSession(null)
    setSessionLoading(false)
    setCompleteResult(null)
    setStockShortfalls(null)
    setError('')
    setConfirmingFinish(false)
    onSelectionReset()
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
    onSelectionReset()
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

  async function handlePause() {
    if (!session || !token) return
    setPausing(true)
    setError('')
    try {
      await pickingApi.pausePicking(token, session.session.id)
      backToOrders()
      await refreshQueue()
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
      onSelectionReset()
      await loadSession(session.session.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reset picking session')
    } finally {
      setResetting(false)
    }
  }

  // handleConfirmComplete is the ONLY place that calls the completePicking
  // API -- the header's "Complete" button just opens the review step
  // (setConfirmingFinish(true)), it never calls this directly.
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
      await refreshQueue()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete picking')
    } finally {
      setCompleting(false)
    }
  }

  return {
    activeOrder, session, sessionLoading, error, setError, loadSession, urlSessionId, backToOrders,
    heldByOther, readOnly, handleTakeOver,
    selectOrder, continueToPacking,
    pausing, handlePause,
    resetConfirmOpen, setResetConfirmOpen, resetting, handleResetSession,
    confirmingFinish, setConfirmingFinish, completing, completeResult, handleConfirmComplete,
    stockShortfalls, setStockShortfalls,
  }
}
