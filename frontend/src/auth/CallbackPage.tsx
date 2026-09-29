import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AuthLayout } from './AuthLayout'
import { useAuth } from './AuthContext'

export function CallbackPage() {
  const { login } = useAuth()
  const { t } = useTranslation('auth')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    const refreshToken = params.get('refresh_token')

    if (!token || !refreshToken) {
      setError(t('callback.noToken'))
      return
    }

    // Basic JWT structure validation before storing
    const parts = token.split('.')
    if (parts.length !== 3) {
      setError(t('callback.malformed'))
      return
    }

    login(token, refreshToken)
    // Remove token from URL before redirecting
    window.history.replaceState({}, '', '/')
    window.location.replace('/')
  }, [login, t])

  if (error) {
    return (
      <AuthLayout title={t('callback.failed')} subtitle={<span role="alert" className="text-danger">{error}</span>}>
        <a href="/" className="text-sm font-medium text-accent hover:underline">{t('callback.returnToLogin')}</a>
      </AuthLayout>
    )
  }

  return <AuthLayout title="CommerceOS" subtitle={t('callback.signingIn')} />
}
