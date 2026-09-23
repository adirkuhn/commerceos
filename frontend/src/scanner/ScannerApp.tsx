import { useState } from 'react'
import { useScannerAuth } from './useScannerAuth'
import { ClaimScreen } from './ClaimScreen'
import { AppLauncher, type ScannerAppID } from './AppLauncher'
import { InventoryApp, type Resolved } from './InventoryApp'
import { PriceLookupApp } from './PriceLookupApp'
import { DiscardApp } from './DiscardApp'
import { PurchaseOrderPickerScreen } from './PurchaseOrderPickerScreen'
import { BookingScreen } from './BookingScreen'
import { SalesOrderPickerScreen } from './SalesOrderPickerScreen'
import { PickingScreen } from './PickingScreen'
import { SalesOrderPackerScreen } from './SalesOrderPackerScreen'
import { PackingScreen } from './PackingScreen'
import type { SalesOrder } from '../oms/types'

// Full-screen kiosk-style app for the handheld scanner device -- bypasses
// the normal Sidebar/Topbar chrome and ../auth/AuthContext entirely (see
// useScannerAuth's doc comment). No router: a small local state machine
// hosts the launcher plus each mini-app's own linear flow, matching the
// rest of this SPA's "no router library" convention.
export default function ScannerApp() {
  const { token, setToken, clearToken } = useScannerAuth()
  const [app, setApp] = useState<ScannerAppID | null>(null)
  const [bookingId, setBookingId] = useState<string | null>(null)
  const [pickingSessionId, setPickingSessionId] = useState<string | null>(null)
  const [pickingOrder, setPickingOrder] = useState<SalesOrder | null>(null)
  const [packingSessionId, setPackingSessionId] = useState<string | null>(null)
  const [packingOrder, setPackingOrder] = useState<SalesOrder | null>(null)
  // Set by Price Lookup's "View in Inventory" -- lets Inventory open
  // straight to this SKU's screen instead of showing its own scan step.
  const [inventoryInitialResolved, setInventoryInitialResolved] = useState<Resolved | null>(null)

  if (!token) {
    return <ClaimScreen onClaimed={setToken} />
  }

  if (!app) {
    return <AppLauncher onSelect={setApp} onLogout={clearToken} />
  }

  if (app === 'inventory') {
    return (
      <InventoryApp
        token={token}
        initialResolved={inventoryInitialResolved ?? undefined}
        onExit={() => { setApp(null); setInventoryInitialResolved(null) }}
      />
    )
  }

  if (app === 'price-lookup') {
    return (
      <PriceLookupApp
        token={token}
        onExit={() => setApp(null)}
        onViewInventory={resolved => { setInventoryInitialResolved(resolved); setApp('inventory') }}
      />
    )
  }

  if (app === 'discard') {
    return <DiscardApp token={token} onExit={() => setApp(null)} />
  }

  if (app === 'pick-order') {
    if (!pickingSessionId || !pickingOrder) {
      return (
        <SalesOrderPickerScreen
          token={token}
          onPickingStarted={(sessionId, order) => { setPickingSessionId(sessionId); setPickingOrder(order) }}
          onLogout={clearToken}
          onBack={() => setApp(null)}
        />
      )
    }
    return (
      <PickingScreen
        token={token}
        sessionId={pickingSessionId}
        order={pickingOrder}
        onExit={() => { setPickingSessionId(null); setPickingOrder(null) }}
        onAuthError={clearToken}
        onContinueToPacking={(sessionId, order) => {
          setPickingSessionId(null); setPickingOrder(null)
          setPackingSessionId(sessionId); setPackingOrder(order); setApp('pack-order')
        }}
      />
    )
  }

  if (app === 'pack-order') {
    if (!packingSessionId || !packingOrder) {
      return (
        <SalesOrderPackerScreen
          token={token}
          onPackingStarted={(sessionId, order) => { setPackingSessionId(sessionId); setPackingOrder(order) }}
          onLogout={clearToken}
          onBack={() => setApp(null)}
        />
      )
    }
    return (
      <PackingScreen
        token={token}
        sessionId={packingSessionId}
        order={packingOrder}
        onExit={() => { setPackingSessionId(null); setPackingOrder(null) }}
        onAuthError={clearToken}
      />
    )
  }

  // Only 'receive-po', 'inventory', 'price-lookup', 'discard', 'pick-order',
  // and 'pack-order' are implemented -- the other tiles in AppLauncher are
  // disabled, so app can't reach this branch with any other value yet.
  if (!bookingId) {
    return (
      <PurchaseOrderPickerScreen
        token={token}
        onBookingStarted={setBookingId}
        onLogout={clearToken}
        onBack={() => setApp(null)}
      />
    )
  }

  return (
    <BookingScreen
      token={token}
      bookingId={bookingId}
      onExit={() => setBookingId(null)}
      onAuthError={clearToken}
    />
  )
}
