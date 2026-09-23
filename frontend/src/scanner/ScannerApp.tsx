import { Navigate, Route, Routes } from 'react-router-dom'
import { useScannerAuth } from './useScannerAuth'
import { ClaimScreen } from './ClaimScreen'
import { AppLauncher, type ScannerAppID } from './AppLauncher'
import { InventoryApp } from './InventoryApp'
import { PriceLookupApp } from './PriceLookupApp'
import { DiscardApp } from './DiscardApp'
import { PurchaseOrderPickerScreen } from './PurchaseOrderPickerScreen'
import { SalesOrderPickerScreen } from './SalesOrderPickerScreen'
import { SalesOrderPackerScreen } from './SalesOrderPackerScreen'
import { BookingRoute, PackSessionRoute, PickOrderRoute, PickSessionRoute } from './sessionRoutes'
import { scannerPaths } from './scannerRoutes'
import { useStackNavigation } from '../routing/useStackNavigation'

const APP_PATHS: Record<ScannerAppID, string> = {
  'receive-po': scannerPaths.receive,
  'inventory': scannerPaths.inventory,
  'price-lookup': scannerPaths.priceLookup,
  'discard': scannerPaths.discard,
  'pick-order': scannerPaths.pick,
  'pack-order': scannerPaths.pack,
}

// Full-screen kiosk-style app for the handheld scanner device -- bypasses
// the normal Sidebar/Topbar chrome and ../auth/AuthContext entirely (see
// useScannerAuth's doc comment). Mounted at /scanner/*: every screen has
// its own URL (see ./scannerRoutes) so a refresh or the device's Back
// button keeps the operator where they were instead of resetting to the
// launcher. Route paths below are relative to /scanner.
// The scanner is dark in every device appearance (dim warehouse floors,
// glare on handhelds), so its subtree always resolves the dark tokens.
export default function ScannerApp() {
  return (
    <div className="dark contents">
      <ScannerAppContent />
    </div>
  )
}

function ScannerAppContent() {
  const { token, setToken, clearToken } = useScannerAuth()
  const { open, exitTo } = useStackNavigation()

  // Unpaired: pair first, then render whatever the URL already points at --
  // a deep link survives the pairing step.
  if (!token) {
    return <ClaimScreen onClaimed={setToken} />
  }

  const toLauncher = () => exitTo(scannerPaths.launcher)

  return (
    <Routes>
      <Route index element={<AppLauncher onSelect={app => open(APP_PATHS[app])} onLogout={clearToken} />} />

      <Route
        path="receive"
        element={<PurchaseOrderPickerScreen token={token} onBookingStarted={id => open(scannerPaths.booking(id))} onLogout={clearToken} onBack={toLauncher} />}
      />
      <Route path="receive/:bookingId" element={<BookingRoute token={token} onAuthError={clearToken} />} />

      <Route path="inventory/*" element={<InventoryApp token={token} onExit={toLauncher} />} />
      <Route path="price-lookup/*" element={<PriceLookupApp token={token} onExit={toLauncher} />} />
      <Route path="discard/*" element={<DiscardApp token={token} onExit={toLauncher} />} />

      <Route
        path="pick"
        element={<SalesOrderPickerScreen token={token} onPickingStarted={id => open(scannerPaths.pickSession(id))} onOrderChosenForList={id => open(scannerPaths.pickOrder(id))} onLogout={clearToken} onBack={toLauncher} />}
      />
      <Route path="pick/order/:orderId" element={<PickOrderRoute token={token} />} />
      <Route path="pick/:sessionId" element={<PickSessionRoute token={token} onAuthError={clearToken} />} />

      <Route
        path="pack"
        element={<SalesOrderPackerScreen token={token} onPackingStarted={id => open(scannerPaths.packSession(id))} onLogout={clearToken} onBack={toLauncher} />}
      />
      <Route path="pack/:sessionId" element={<PackSessionRoute token={token} onAuthError={clearToken} />} />

      <Route path="*" element={<Navigate to={scannerPaths.launcher} replace />} />
    </Routes>
  )
}
