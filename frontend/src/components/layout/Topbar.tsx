import { useState } from 'react'
import { Search, Sun, Moon, Monitor, Command, QrCode } from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { PairScannerModal } from '../../scanner/PairScannerModal'
import NotificationBell from './NotificationBell'
import type { ThemePreference } from '../../theme/themePreference'

const APPEARANCE: Record<ThemePreference, { label: string; icon: React.ElementType }> = {
  system: { label: 'System', icon: Monitor },
  light: { label: 'Light', icon: Sun },
  dark: { label: 'Dark', icon: Moon },
}

const NAV_TITLES: Record<string, string> = {
  dashboard: 'Dashboard',
  'sales-orders': 'Sales Order',
  customers: 'Customer',
  products: 'Products · PIM',
  inventory: 'Inventory · WMS',
  orders: 'Orders · OMS',
  picking: 'Picking & Packing',
  suppliers: 'Suppliers',
  procurement: 'Procurement',
  integrations: 'Integrations',
  shipping: 'Shipping',
  reports: 'Reports',
  users: 'Users & Permissions',
  settings: 'Settings',
}

export default function Topbar() {
  const { activeNav, themePreference, cycleThemePreference, setCommandOpen } = useApp()
  const appearance = APPEARANCE[themePreference]
  const AppearanceIcon = appearance.icon
  const { token } = useAuth()
  const [pairScannerOpen, setPairScannerOpen] = useState(false)

  return (
    <header className="h-14 shrink-0 flex items-center gap-4 px-5 border-b border-line bg-surface">
      <h1 className="text-sm font-semibold text-ink mr-2">
        {NAV_TITLES[activeNav] ?? activeNav}
      </h1>

      {/* Search */}
      <button
        onClick={() => setCommandOpen(true)}
        className="flex-1 max-w-sm flex items-center gap-2 px-3 py-1.5 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-4 text-xs hover:bg-hover transition-colors"
      >
        <Search size={13} />
        <span className="flex-1 text-left">Search everything...</span>
        <span className="flex items-center gap-0.5 bg-surface px-1.5 py-0.5 rounded text-2xs text-ink-4 font-mono">
          <Command size={10} />K
        </span>
      </button>

      <div className="flex items-center gap-2 ml-auto">
        {/* Pair a handheld scanner -- global, not tied to any one page */}
        <button
          onClick={() => setPairScannerOpen(true)}
          title="Pair Scanner"
          aria-label="Pair Scanner"
          className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-hover transition-colors"
        >
          <QrCode size={16} className="text-ink-3" />
        </button>

        {/* Notifications */}
        <NotificationBell />

        {/* Appearance: follows the OS by default, click to override */}
        <button
          onClick={cycleThemePreference}
          aria-label={`Appearance: ${appearance.label}`}
          title={`Appearance: ${appearance.label}`}
          className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-hover transition-colors"
        >
          <AppearanceIcon size={16} className="text-ink-3" />
        </button>

        {/* Status pill */}
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full dark:bg-green-500/10 bg-green-50 border dark:border-green-500/20 border-green-200">
          <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
          <span className="text-2xs font-medium text-success">All systems live</span>
        </div>
      </div>

      {pairScannerOpen && token && (
        <PairScannerModal token={token} onClose={() => setPairScannerOpen(false)} />
      )}
    </header>
  )
}
