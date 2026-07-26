import { AuthProvider, useAuth } from './auth/AuthContext'
import { LoginPage } from './auth/LoginPage'
import { CallbackPage } from './auth/CallbackPage'
import { AppProvider, useApp } from './context/AppContext'
import Sidebar from './components/layout/Sidebar'
import Topbar from './components/layout/Topbar'
import CommandPalette from './components/layout/CommandPalette'
import Dashboard from './pages/Dashboard'
import Products from './pages/Products'
import Inventory from './pages/Inventory'
import Orders from './pages/Orders'
import Picking from './pages/Picking'
import Integrations from './pages/Integrations'
import Users from './pages/Users'
import Suppliers from './pages/Suppliers'
import Procurement from './pages/Procurement'
import Shipping from './pages/Shipping'
import Reports from './pages/Reports'
import Settings from './pages/Settings'

const CALLBACK_PATH = (import.meta.env.VITE_LM2_CALLBACK_PATH as string) ?? '/auth/callback'

function AppContent() {
  const { activeNav } = useApp()

  const renderPage = () => {
    switch (activeNav) {
      case 'dashboard': return <Dashboard />
      case 'products': return <Products />
      case 'inventory': return <Inventory />
      case 'orders': return <Orders />
      case 'picking': return <Picking />
      case 'integrations': return <Integrations />
      case 'users': return <Users />
      case 'suppliers': return <Suppliers />
      case 'procurement': return <Procurement />
      case 'shipping': return <Shipping />
      case 'reports': return <Reports />
      case 'settings': return <Settings />
      default: return <Dashboard />
    }
  }

  return (
    <div className="flex h-screen dark:bg-[#0B1220] bg-[#F9FAFB] overflow-hidden">
      <Sidebar />
      <div className="flex flex-col flex-1 overflow-hidden">
        <Topbar />
        <main className="flex-1 overflow-hidden dark:bg-[#0B1220] bg-[#F9FAFB]">
          {renderPage()}
        </main>
      </div>
      <CommandPalette />
    </div>
  )
}

function AppShell() {
  const { isAuthenticated } = useAuth()
  const isCallback = window.location.pathname === CALLBACK_PATH

  if (isCallback) return <CallbackPage />
  if (!isAuthenticated) return <LoginPage />

  return (
    <AppProvider>
      <AppContent />
    </AppProvider>
  )
}

export function App() {
  return (
    <AuthProvider>
      <AppShell />
    </AuthProvider>
  )
}
