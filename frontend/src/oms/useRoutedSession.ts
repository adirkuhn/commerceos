import i18next from 'i18next'
import { useCallback, useEffect, useRef } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import * as omsApi from './api'
import { idFromPath } from '../routes'
import { useStackNavigation } from '../routing/useStackNavigation'
import type { SalesOrder } from './types'

interface SessionDetailShape {
  session: { id: string; sales_order_id: string }
}

interface RoutedSessionOptions<D extends SessionDetailShape> {
  // List page path, e.g. "/picking"; a session lives at `${basePath}/:id`.
  basePath: string
  token: string | null
  loadDetail: (token: string, sessionId: string) => Promise<D>
  // The order already on screen, if any -- skips refetching it when the
  // loaded session belongs to it.
  currentOrder: SalesOrder | null
  onLoading: () => void
  onLoaded: (order: SalesOrder, detail: D) => void
  onError: (message: string) => void
  onCleared: () => void
}

export interface RoutedSession {
  urlSessionId: string | null
  // After startOrResume: show that session (pushes its URL, or replaces
  // when switching from one session to another). Returns true when the
  // session is already the one in the URL -- the caller reloads it itself,
  // since the URL (and so the loading effect) doesn't change.
  showSession: (sessionId: string) => boolean
  backToList: () => void
}

// Picking/Packing pages: the URL decides which session is open. Entering
// `${basePath}/:sessionId` (click, refresh, Back/Forward) loads it
// read-only -- get session, then its order -- never startOrResume, which
// would fail or open a new session once the order moved on. Leaving the
// URL clears the panel.
export function useRoutedSession<D extends SessionDetailShape>(opts: RoutedSessionOptions<D>): RoutedSession {
  const { basePath, token } = opts
  const navigate = useNavigate()
  const location = useLocation()
  const { open, exitTo } = useStackNavigation()
  const urlSessionId = idFromPath(location.pathname, basePath)

  // Latest callbacks/order, read by the effect without re-running it --
  // it is keyed on the URL alone.
  const latest = useRef(opts)
  latest.current = opts

  useEffect(() => {
    const o = latest.current
    if (!token) return
    if (!urlSessionId) {
      o.onCleared()
      return
    }
    let current = true
    o.onLoading()
    ;(async () => {
      try {
        const detail = await o.loadDetail(token, urlSessionId)
        const known = latest.current.currentOrder
        const order = known?.id === detail.session.sales_order_id
          ? known
          : (await omsApi.getSalesOrder(token, detail.session.sales_order_id)).sales_order
        if (current) latest.current.onLoaded(order, detail)
      } catch (err) {
        if (current) latest.current.onError(err instanceof Error ? err.message : i18next.t('orders:errors.loadSession'))
      }
    })()
    return () => { current = false }
  }, [urlSessionId, token])

  const sessionPath = useCallback((id: string) => `${basePath}/${encodeURIComponent(id)}`, [basePath])

  const showSession = useCallback((sessionId: string) => {
    if (sessionId === urlSessionId) return true
    if (urlSessionId) {
      // Switching sessions from inside one: replace, so Back goes to the
      // list rather than through every order clicked along the way.
      navigate(sessionPath(sessionId), { replace: true, state: location.state })
    } else {
      open(sessionPath(sessionId))
    }
    return false
  }, [urlSessionId, navigate, sessionPath, location.state, open])

  const backToList = useCallback(() => {
    // With no session in the URL (a start that failed) there is nothing to
    // navigate away from -- just clear.
    if (urlSessionId) exitTo(basePath)
    else latest.current.onCleared()
  }, [urlSessionId, exitTo, basePath])

  return { urlSessionId, showSession, backToList }
}
