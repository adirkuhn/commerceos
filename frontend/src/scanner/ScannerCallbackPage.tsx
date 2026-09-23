import { useEffect, useState } from 'react'
import { storeScannerSession } from './useScannerAuth'

// Google-login counterpart to ClaimScreen's pairing-code flow -- lands here
// after lm2's OAuth round trip (see ClaimScreen's "Sign in with Google"
// button), same query-param contract ../auth/CallbackPage uses, but writes
// into the scanner's own storage (storeScannerSession) instead of
// ../auth/AuthContext's, and returns to /scanner instead of /.
export function ScannerCallbackPage() {
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    const refreshToken = params.get('refresh_token')

    if (!token || !refreshToken) {
      setError('No token received from authentication server.')
      return
    }
    if (token.split('.').length !== 3) {
      setError('Malformed token received.')
      return
    }

    storeScannerSession(token, refreshToken)
    window.location.replace('/scanner')
  }, [])

  if (error) {
    return (
      <div className="fixed inset-0 flex items-center justify-center bg-night-950 p-6 text-center">
        <div>
          <p className="text-sm text-red-400">{error}</p>
          <a href="/scanner" className="mt-2 inline-block text-sm text-blue-400">Back to pairing</a>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-night-950">
      <p className="text-sm text-gray-300">Signing you in…</p>
    </div>
  )
}
