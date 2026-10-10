import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useRoutedSession } from '../oms/useRoutedSession'
import { useSessionLock } from '../fulfillment/useSessionLock'
import type { SalesOrder } from '../oms/types'
import * as packingApi from '../scanner/api'
import type { PackingSessionDetail } from '../scanner/types'
import { apiErrorText } from '../api/apiErrorText'

export const PACKING_PATH = '/packing'

interface PackingSessionOptions {
  token: string | null
  me: string
  // Picking a new order from the queue -- the page drops per-order view state.
  onSelect: () => void
  // The session left the screen (URL cleared, Back) -- drop everything.
  onCleared: () => void
}

// Which order/session is open on the Packing page, its loading/error state
// and its lock. Each open session has its own URL (/packing/:sessionId) --
// see useRoutedSession. ?order=<id> is kept as an action URL (Picking's
// "Continue to packing"): it starts/resumes that order's session, then
// replaces itself with the session URL.
export function usePackingSession({ token, me, onSelect, onCleared }: PackingSessionOptions) {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  const [activeOrder, setActiveOrder] = useState<SalesOrder | null>(null)
  const [session, setSession] = useState<PackingSessionDetail | null>(null)
  const [sessionLoading, setSessionLoading] = useState(false)
  const [error, setError] = useState('')

  const loadSession = useCallback(async (sessionId: string) => {
    if (!token) return
    try {
      setSession(await packingApi.getPackingSession(token, sessionId))
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('packing:errors.loadSession'))
    } finally {
      setSessionLoading(false)
    }
  }, [token])

  // Opening an order only views it; the first action takes the lock. While
  // we hold it we heartbeat it, and leaving the order releases it.
  const { heldByOther } = useSessionLock({
    sessionId: session?.session.id ?? null, me,
    lockedBy: session?.session.locked_by, lockedAt: session?.session.locked_at,
    active: !!session && session.session.status !== 'completed',
    touch: id => packingApi.touchPackingSession(token ?? '', id),
    release: id => packingApi.releasePackingSessionLock(token ?? '', id),
    onLost: () => setError(i18next.t('packing:errors.takenOver')),
  })

  async function takeOver() {
    if (!session || !token) return
    setError('')
    try {
      await packingApi.takeOverPackingSession(token, session.session.id)
      await loadSession(session.session.id)
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('packing:errors.takeOver'))
    }
  }

  // Claims the lock (the heartbeat takes it when free) before working the
  // order outside a scan, e.g. making its label. False when someone else
  // holds it -- the error is already shown.
  async function claim(): Promise<boolean> {
    if (!session || !token) return true
    try {
      await packingApi.touchPackingSession(token, session.session.id)
      await loadSession(session.session.id)
      return true
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('packing:errors.someoneElse'))
      return false
    }
  }

  function clearSelection() {
    setActiveOrder(null)
    setSession(null)
    setSessionLoading(false)
    setError('')
    onCleared()
  }

  const { urlSessionId, showSession, backToList } = useRoutedSession({
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
    onSelect()
    setActiveOrder(order)
    setSession(null)
    setSessionLoading(true)
    try {
      const newSession = await packingApi.startOrResumePacking(token ?? '', order.id)
      if (showSession(newSession.id)) await loadSession(newSession.id)
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('packing:errors.start'))
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
        setError(apiErrorText(err) ?? i18next.t('packing:errors.start'))
        setSessionLoading(false)
      })
    return () => { current = false }
  }, [orderToStart, token, navigate])

  return {
    activeOrder, session, sessionLoading, error, setError, loadSession,
    selectOrder, backToOrders: backToList, urlSessionId, orderToStart,
    heldByOther, readOnly: heldByOther !== null, takeOver, claim,
  }
}
