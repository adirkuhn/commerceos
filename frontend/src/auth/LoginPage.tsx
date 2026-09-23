import { useState } from 'react'
import { hasLoginUrl, APP_ID, lm2Origin } from './loginConfig'
import { AuthLayout, primaryButtonClass, secondaryButtonClass } from './AuthLayout'

const LM2_LOGIN_URL = import.meta.env.VITE_LM2_LOGIN_URL as string
const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'

function ConfigMissing() {
  return (
    <AuthLayout
      title="CommerceOS"
      subtitle={<>The lm2 login server is not configured. Set <code>VITE_LM2_LOGIN_URL</code> to enable sign-in.</>}
    />
  )
}

export function LoginPage() {
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
      if (!res.ok) throw new Error(body.error?.message ?? body.message ?? 'Google sign-in is unavailable')
      window.location.href = body.data.url
    } catch (e) {
      setGoogleError(e instanceof Error ? e.message : 'Google sign-in is unavailable')
      setGoogleLoading(false)
    }
  }

  return (
    <AuthLayout title="CommerceOS" subtitle="Sign in to continue">
      <div className="space-y-3">
        <button onClick={handleLogin} className={primaryButtonClass}>
          Sign in with lm2
        </button>
        <button onClick={handleGoogleLogin} disabled={googleLoading} className={secondaryButtonClass}>
          {googleLoading ? 'Redirecting…' : 'Sign in with Google'}
        </button>
        {googleError && <p role="alert" className="text-xs text-danger">{googleError}</p>}
      </div>
    </AuthLayout>
  )
}
