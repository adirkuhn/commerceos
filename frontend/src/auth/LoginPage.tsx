
const LM2_LOGIN_URL = import.meta.env.VITE_LM2_LOGIN_URL as string
const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'

export function LoginPage() {
  const handleLogin = () => {
    const callbackUrl = `${window.location.origin}${CALLBACK_PATH}`
    const redirectTo = `${LM2_LOGIN_URL}?redirect=${encodeURIComponent(callbackUrl)}`
    window.location.href = redirectTo
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
      </div>
    </div>
  )
}
