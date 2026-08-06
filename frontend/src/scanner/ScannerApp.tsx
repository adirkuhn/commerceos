import { useState } from 'react'
import { useScannerAuth } from './useScannerAuth'
import { ClaimScreen } from './ClaimScreen'
import { PurchaseOrderPickerScreen } from './PurchaseOrderPickerScreen'
import { BookingScreen } from './BookingScreen'

// Full-screen kiosk-style app for the handheld scanner device -- bypasses
// the normal Sidebar/Topbar chrome and ../auth/AuthContext entirely (see
// useScannerAuth's doc comment). No router: a tiny local state machine is
// enough for a three-screen linear flow, matching the rest of this SPA's
// "no router library" convention.
export default function ScannerApp() {
  const { token, setToken, clearToken } = useScannerAuth()
  const [bookingId, setBookingId] = useState<string | null>(null)

  if (!token) {
    return <ClaimScreen onClaimed={setToken} />
  }

  if (!bookingId) {
    return (
      <PurchaseOrderPickerScreen
        token={token}
        onBookingStarted={setBookingId}
        onLogout={clearToken}
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
