import { BrowserRouter, Routes, Route, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth/AuthContext'
import { LoginPage } from './auth/LoginPage'
import { CallbackPage } from './auth/CallbackPage'
import { AcceptInvite } from './auth/AcceptInvite'
import ScannerApp from './scanner/ScannerApp'
import { ScannerCallbackPage } from './scanner/ScannerCallbackPage'
import { AppProvider } from './context/AppContext'
import Sidebar from './components/layout/Sidebar'
import Topbar from './components/layout/Topbar'
import CommandPalette from './components/layout/CommandPalette'
import { PAGE_ROUTES } from './routes'

const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'
const SCANNER_PATH = '/scanner'
const SCANNER_CALLBACK_PATH = '/scanner/auth/callback'
const ACCEPT_INVITE_PATH = '/accept-invite'

function NotFoundPage() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center">
        <p className="text-sm font-medium dark:text-gray-200 text-gray-800">Page not found</p>
        <p className="text-xs dark:text-gray-500 text-gray-400 mt-1">This page doesn&apos;t exist.</p>
      </div>
    </div>
  )
}

function AppContent() {
  return (
    <div className="flex h-screen dark:bg-[#0B1220] bg-[#F9FAFB] overflow-hidden">
      <Sidebar />
      <div className="flex flex-col flex-1 overflow-hidden">
        <Topbar />
        <main className="flex-1 overflow-y-auto dark:bg-[#0B1220] bg-[#F9FAFB]">
          <Routes>
            {PAGE_ROUTES.map(({ path, Component }) => (
              <Route key={path} path={path} element={<Component />} />
            ))}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </main>
      </div>
      <CommandPalette />
    </div>
  )
}

export function AppShell() {
  const { isAuthenticated } = useAuth()
  const location = useLocation()
  const isCallback = location.pathname === CALLBACK_PATH
  const isScanner = location.pathname === SCANNER_PATH
  const isScannerCallback = location.pathname === SCANNER_CALLBACK_PATH
  const isAcceptInvite = location.pathname === ACCEPT_INVITE_PATH

  // The scanner device pairs with its own token (see
  // ./scanner/useScannerAuth) and never goes through ../auth/AuthContext --
  // checked before isAuthenticated so a scanner device with no admin session
  // still reaches the pairing screen instead of LoginPage. Checked before
  // isScanner since SCANNER_CALLBACK_PATH is a distinct path, not a prefix
  // match on SCANNER_PATH.
  if (isScannerCallback) return <ScannerCallbackPage />
  if (isScanner) return <ScannerApp />
  if (isCallback) return <CallbackPage />
  // An invitee following an email link has no session yet -- same reasoning
  // as isCallback/isScanner, checked before the isAuthenticated gate below.
  if (isAcceptInvite) return <AcceptInvite />
  if (!isAuthenticated) return <LoginPage />

  return (
    <AppProvider>
      <AppContent />
    </AppProvider>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </BrowserRouter>
  )
}
