import { useEffect, useState } from 'react'
import { AuthLayout } from './AuthLayout'
import { useAuth } from './AuthContext'

export function CallbackPage() {
  const { login } = useAuth()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    const refreshToken = params.get('refresh_token')

    if (!token || !refreshToken) {
      setError('No token received from authentication server.')
      return
    }

    // Basic JWT structure validation before storing
    const parts = token.split('.')
    if (parts.length !== 3) {
      setError('Malformed token received.')
      return
    }

    login(token, refreshToken)
    // Remove token from URL before redirecting
    window.history.replaceState({}, '', '/')
    window.location.replace('/')
  }, [login])

  if (error) {
    return (
      <AuthLayout title="Sign-in failed" subtitle={<span role="alert" className="text-danger">{error}</span>}>
        <a href="/" className="text-sm font-medium text-accent hover:underline">Return to login</a>
      </AuthLayout>
    )
  }

  return <AuthLayout title="CommerceOS" subtitle="Signing you in…" />
}
