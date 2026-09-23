import { useNavigate, useParams } from 'react-router-dom'
import { useStackNavigation } from '../routing/useStackNavigation'
import * as api from './api'
import * as omsApi from '../oms/api'
import { useResource } from '../routing/useResource'
import { ScannerLoadGate } from './ScannerLoadGate'
import { PickingScreen } from './PickingScreen'
import { PackingScreen } from './PackingScreen'
import { BookingScreen } from './BookingScreen'
import { scannerPaths } from './scannerRoutes'
import type { SalesOrder } from '../oms/types'

// URL -> screen adapters for the session-based scanner flows. Each reads the
// session id from the path and reloads what its screen needs, so /pick/:id,
// /pack/:id and /receive/:id survive a refresh. Deliberately read-only
// loads (get session -> get its order): re-calling startOrResume on refresh
// would open a brand-new session once the old one is completed.

interface SessionRouteProps {
  token: string
  onAuthError: () => void
}

async function loadSessionOrder(token: string, salesOrderId: string): Promise<SalesOrder> {
  return (await omsApi.getSalesOrder(token, salesOrderId)).sales_order
}

export function PickSessionRoute({ token, onAuthError }: SessionRouteProps) {
  const { sessionId = '' } = useParams()
  const navigate = useNavigate()
  const { exitTo } = useStackNavigation()
  const order = useResource(async () => {
    const detail = await api.getPickingSession(token, sessionId)
    return loadSessionOrder(token, detail.session.sales_order_id)
  }, [token, sessionId])

  return (
    <ScannerLoadGate resource={order} onBack={() => exitTo(scannerPaths.pick)}>
      {loaded => (
        <PickingScreen
          key={sessionId}
          token={token}
          sessionId={sessionId}
          order={loaded}
          onExit={() => exitTo(scannerPaths.pick)}
          onAuthError={onAuthError}
          // replace: Back from packing should return to the order list, not
          // to a picking session that is already completed.
          onContinueToPacking={packingSessionId => navigate(scannerPaths.packSession(packingSessionId), { replace: true })}
        />
      )}
    </ScannerLoadGate>
  )
}

export function PackSessionRoute({ token, onAuthError }: SessionRouteProps) {
  const { sessionId = '' } = useParams()
  const { exitTo } = useStackNavigation()
  const order = useResource(async () => {
    const detail = await api.getPackingSession(token, sessionId)
    return loadSessionOrder(token, detail.session.sales_order_id)
  }, [token, sessionId])

  return (
    <ScannerLoadGate resource={order} onBack={() => exitTo(scannerPaths.pack)}>
      {loaded => (
        <PackingScreen
          key={sessionId}
          token={token}
          sessionId={sessionId}
          order={loaded}
          onExit={() => exitTo(scannerPaths.pack)}
          onAuthError={onAuthError}
        />
      )}
    </ScannerLoadGate>
  )
}

// BookingScreen already loads everything from bookingId itself -- this only
// maps the URL param in and the exit back out.
export function BookingRoute({ token, onAuthError }: SessionRouteProps) {
  const { bookingId = '' } = useParams()
  const { exitTo } = useStackNavigation()
  return (
    <BookingScreen
      key={bookingId}
      token={token}
      bookingId={bookingId}
      onExit={() => exitTo(scannerPaths.receive)}
      onAuthError={onAuthError}
    />
  )
}
