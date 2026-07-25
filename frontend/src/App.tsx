import { AuthProvider, useAuth } from './auth/AuthContext'
import { LoginPage } from './auth/LoginPage'
import { CallbackPage } from './auth/CallbackPage'

const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'

function AppShell() {
  const { isAuthenticated, logout } = useAuth()
  const isCallback = window.location.pathname === CALLBACK_PATH

  if (isCallback) return <CallbackPage />
  if (!isAuthenticated) return <LoginPage />

  return (
    <div>
      <header style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '12px 24px',
        borderBottom: '1px solid #eee',
      }}>
        <strong>CommerceOS</strong>
        <button onClick={logout} style={{ background: 'none', border: '1px solid #ccc', padding: '6px 12px', cursor: 'pointer', borderRadius: '4px' }}>
          Sign out
        </button>
      </header>
      <main style={{ padding: '24px' }}>
        {/* IAM module and other modules mount here */}
        <p>Welcome to CommerceOS.</p>
      </main>
    </div>
  )
}

export function App() {
  return (
    <AuthProvider>
      <AppShell />
    </AuthProvider>
  )
}
