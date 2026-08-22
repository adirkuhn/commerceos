import { useState } from 'react'
import { hasLoginUrl, APP_ID, lm2Origin } from './loginConfig'

const LM2_LOGIN_URL = import.meta.env.VITE_LM2_LOGIN_URL as string
const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'

function ConfigMissing() {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: '#f5f5f5',
    }}>
      <div style={{
        background: '#fff',
        padding: '48px',
        borderRadius: '8px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
        textAlign: 'center',
        maxWidth: '420px',
      }}>
        <h1 style={{ margin: '0 0 8px', fontSize: '1.5rem' }}>CommerceOS</h1>
        <p style={{ margin: 0, color: '#666' }}>
          The lm2 login server is not configured. Set <code>VITE_LM2_LOGIN_URL</code> to enable sign-in.
        </p>
      </div>
    </div>
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
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: '#f5f5f5',
    }}>
      <div style={{
        background: '#fff',
        padding: '48px',
        borderRadius: '8px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
        textAlign: 'center',
        minWidth: '320px',
      }}>
        <h1 style={{ margin: '0 0 8px', fontSize: '1.5rem' }}>CommerceOS</h1>
        <p style={{ margin: '0 0 32px', color: '#666' }}>Sign in to continue</p>
        <button
          onClick={handleLogin}
          style={{
            width: '100%',
            padding: '12px',
            background: '#1a1a1a',
            color: '#fff',
            border: 'none',
            borderRadius: '6px',
            fontSize: '1rem',
            cursor: 'pointer',
          }}
        >
          Sign in with lm2
        </button>

        {googleError && <p style={{ color: 'red', margin: '12px 0 0', fontSize: '0.85rem' }}>{googleError}</p>}

        <button
          onClick={handleGoogleLogin}
          disabled={googleLoading}
          style={{
            width: '100%',
            marginTop: '12px',
            padding: '12px',
            background: '#fff',
            color: '#1a1a1a',
            border: '1px solid #ddd',
            borderRadius: '6px',
            fontSize: '1rem',
            cursor: 'pointer',
          }}
        >
          {googleLoading ? 'Redirecting…' : 'Sign in with Google'}
        </button>
      </div>
    </div>
  )
}
