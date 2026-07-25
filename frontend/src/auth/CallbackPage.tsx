import { useEffect, useState } from 'react'
import { useAuth } from './AuthContext'

export function CallbackPage() {
  const { login } = useAuth()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')

    if (!token) {
      setError('No token received from authentication server.')
      return
    }

    // Basic JWT structure validation before storing
    const parts = token.split('.')
    if (parts.length !== 3) {
      setError('Malformed token received.')
      return
    }

    login(token)
    // Remove token from URL before redirecting
    window.history.replaceState({}, '', '/')
    window.location.replace('/')
  }, [login])

  if (error) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div style={{ textAlign: 'center' }}>
          <p style={{ color: 'red' }}>{error}</p>
          <a href="/">Return to login</a>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
      <p>Signing you in...</p>
    </div>
  )
}
