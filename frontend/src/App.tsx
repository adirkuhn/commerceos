import { BrowserRouter, Routes, Route, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AuthProvider, useAuth } from './auth/AuthContext'
import { LoginPage } from './auth/LoginPage'
import { CallbackPage } from './auth/CallbackPage'
import { AcceptInvite } from './auth/AcceptInvite'
import ScannerApp from './scanner/ScannerApp'
import { isScannerPath, SCANNER_BASE } from './scanner/scannerRoutes'
import { ScannerCallbackPage } from './scanner/ScannerCallbackPage'
import { AppProvider } from './context/AppContext'
import { DateFormatProvider } from './format/DateFormatProvider'
import { CompanyLanguage } from './i18n/CompanyLanguage'
import Sidebar from './components/layout/Sidebar'
import Topbar from './components/layout/Topbar'
import CommandPalette from './components/layout/CommandPalette'
import { PAGE_ROUTES, SUB_PAGE_ROUTES } from './routes'
import { useApp } from './context/AppContext'
import { GO_TO_SHORTCUTS } from './components/layout/navConfig'
import { useGoToShortcuts } from './shortcuts/useGoToShortcuts'

const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'
const SCANNER_CALLBACK_PATH = '/scanner/auth/callback'
const ACCEPT_INVITE_PATH = '/accept-invite'

function NotFoundPage() {
  const { t } = useTranslation('ui')
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center">
        <p className="text-sm font-medium text-ink">{t('app.pageNotFound')}</p>
        <p className="text-xs text-ink-4 mt-1">{t('app.pageNotFoundHint')}</p>
      </div>
    </div>
  )
}

function GoToShortcuts() {
  const { setActiveNav } = useApp()
  useGoToShortcuts(GO_TO_SHORTCUTS, setActiveNav)
  return null
}

function AppContent() {
  return (
    <div className="flex h-screen bg-canvas overflow-hidden">
      <Sidebar />
      <div className="flex flex-col flex-1 overflow-hidden">
        <Topbar />
        <main className="flex-1 overflow-y-auto bg-canvas">
          <Routes>
            {PAGE_ROUTES.map(({ path, Component }) => (
              <Route key={path} path={path} element={<Component />} />
            ))}
            {SUB_PAGE_ROUTES.map(({ path, Component }) => (
              <Route key={path} path={path} element={<Component />} />
            ))}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </main>
      </div>
      <CommandPalette />
      <GoToShortcuts />
    </div>
  )
}

export function AppShell() {
  const { isAuthenticated, token } = useAuth()
  const location = useLocation()
  const isCallback = location.pathname === CALLBACK_PATH
  const isScanner = isScannerPath(location.pathname)
  const isScannerCallback = location.pathname === SCANNER_CALLBACK_PATH
  const isAcceptInvite = location.pathname === ACCEPT_INVITE_PATH

  // The scanner device pairs with its own token (see
  // ./scanner/useScannerAuth) and never goes through ../auth/AuthContext --
  // checked before isAuthenticated so a scanner device with no admin session
  // still reaches the pairing screen instead of LoginPage. Checked before
  // isScanner since SCANNER_CALLBACK_PATH also sits under /scanner/.
  // ScannerApp owns every other /scanner/* URL (see ./scanner/scannerRoutes).
  if (isScannerCallback) return <ScannerCallbackPage />
  if (isScanner) {
    return (
      <Routes>
        <Route path={`${SCANNER_BASE}/*`} element={<ScannerApp />} />
      </Routes>
    )
  }
  if (isCallback) return <CallbackPage />
  // An invitee following an email link has no session yet -- same reasoning
  // as isCallback/isScanner, checked before the isAuthenticated gate below.
  if (isAcceptInvite) return <AcceptInvite />
  if (!isAuthenticated) return <LoginPage />

  return (
    <AppProvider>
      <CompanyLanguage token={token ?? ''} />
      <DateFormatProvider token={token ?? ''}>
        <AppContent />
      </DateFormatProvider>
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
