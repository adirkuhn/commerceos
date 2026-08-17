import { useState } from 'react'
import { useScannerAuth } from './useScannerAuth'
import { ClaimScreen } from './ClaimScreen'
import { AppLauncher, type ScannerAppID } from './AppLauncher'
import { InventoryApp, type Resolved } from './InventoryApp'
import { PriceLookupApp } from './PriceLookupApp'
import { DiscardApp } from './DiscardApp'
import { PurchaseOrderPickerScreen } from './PurchaseOrderPickerScreen'
import { BookingScreen } from './BookingScreen'

// Full-screen kiosk-style app for the handheld scanner device -- bypasses
// the normal Sidebar/Topbar chrome and ../auth/AuthContext entirely (see
// useScannerAuth's doc comment). No router: a small local state machine
// hosts the launcher plus each mini-app's own linear flow, matching the
// rest of this SPA's "no router library" convention.
export default function ScannerApp() {
  const { token, setToken, clearToken } = useScannerAuth()
  const [app, setApp] = useState<ScannerAppID | null>(null)
  const [bookingId, setBookingId] = useState<string | null>(null)
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

  // Only 'receive-po', 'inventory', 'price-lookup', and 'discard' are
  // implemented -- the other tiles in AppLauncher are disabled, so app
  // can't reach this branch with any other value yet.
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
