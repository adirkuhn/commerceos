import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { hasLoginUrl, APP_ID, lm2Origin } from './loginConfig'
import { AuthLayout, primaryButtonClass, secondaryButtonClass } from './AuthLayout'

const LM2_LOGIN_URL = import.meta.env.VITE_LM2_LOGIN_URL as string
const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'

function ConfigMissing() {
  const { t } = useTranslation('auth')
  return (
    <AuthLayout
      title="CommerceOS"
      subtitle={<Trans t={t} i18nKey="login.notConfigured" values={{ variable: 'VITE_LM2_LOGIN_URL' }} components={{ code: <code /> }} />}
    />
  )
}

export function LoginPage() {
  const { t } = useTranslation('auth')
  const [googleError, setGoogleError] = useState<string | null>(null)
  const [googleLoading, setGoogleLoading] = useState(false)

  if (!hasLoginUrl(import.meta.env)) {
    return <ConfigMissing />
  }

  const handleLogin = () => {
    const callbackUrl = `${window.location.origin}${CALLBACK_PATH}`
    const redirectTo = `${LM2_LOGIN_URL}?app=${APP_ID}&redirect=${encodeURIComponent(callbackUrl)}`
    window.location.href = redirectTo
  }

  // Skips landing on lm2's own AppLoginPage first -- goes straight into the
  // Google consent screen, same direct-frontend-to-lm2 pattern the token
  // refresh call already uses.
  const handleGoogleLogin = async () => {
    setGoogleError(null)
    setGoogleLoading(true)
    try {
      const callbackUrl = `${window.location.origin}${CALLBACK_PATH}`
      const res = await fetch(`${lm2Origin()}/api/v1/auth/app/google/url?app=${APP_ID}&redirect=${encodeURIComponent(callbackUrl)}`)
      const body = await res.json()
      if (!res.ok) throw new Error(body.error?.message ?? body.message ?? t('login.googleUnavailable'))
      window.location.href = body.data.url
    } catch (e) {
      setGoogleError(e instanceof Error ? e.message : t('login.googleUnavailable'))
      setGoogleLoading(false)
    }
  }

  return (
    <AuthLayout title="CommerceOS" subtitle={t('login.subtitle')}>
      <div className="space-y-3">
        <button onClick={handleLogin} className={primaryButtonClass}>
          {t('login.withLm2')}
        </button>
        <button onClick={handleGoogleLogin} disabled={googleLoading} className={secondaryButtonClass}>
          {googleLoading ? t('login.redirecting') : t('login.withGoogle')}
        </button>
        {googleError && <p role="alert" className="text-xs text-danger">{googleError}</p>}
      </div>
    </AuthLayout>
  )
}
