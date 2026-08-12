import { useState } from 'react'
import { Search, Sun, Moon, Command, QrCode } from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { PairScannerModal } from '../../scanner/PairScannerModal'
import NotificationBell from './NotificationBell'

const NAV_TITLES: Record<string, string> = {
  dashboard: 'Dashboard',
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
  const { activeNav, theme, toggleTheme, setCommandOpen } = useApp()
  const { token } = useAuth()
  const [pairScannerOpen, setPairScannerOpen] = useState(false)

  return (
    <header className="h-14 shrink-0 flex items-center gap-4 px-5 border-b dark:border-[#1F2937] border-gray-200 dark:bg-[#111827] bg-white">
      <h1 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mr-2">
        {NAV_TITLES[activeNav] ?? activeNav}
      </h1>

      {/* Search */}
      <button
        onClick={() => setCommandOpen(true)}
        className="flex-1 max-w-sm flex items-center gap-2 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-gray-100 dark:text-gray-500 text-gray-400 text-xs hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
      >
        <Search size={13} />
        <span className="flex-1 text-left">Search everything...</span>
        <span className="flex items-center gap-0.5 dark:bg-[#111827] bg-white px-1.5 py-0.5 rounded text-[10px] dark:text-gray-600 text-gray-400 font-mono">
          <Command size={10} />K
        </span>
      </button>

      <div className="flex items-center gap-2 ml-auto">
        {/* Pair a handheld scanner -- global, not tied to any one page */}
        <button
          onClick={() => setPairScannerOpen(true)}
          title="Pair Scanner"
          className="w-8 h-8 flex items-center justify-center rounded-lg dark:hover:bg-[#1F2937] hover:bg-gray-100 transition-colors"
        >
          <QrCode size={16} className="dark:text-gray-400 text-gray-500" />
        </button>

        {/* Notifications */}
        <NotificationBell />

        {/* Theme toggle */}
        <button
          onClick={toggleTheme}
          className="w-8 h-8 flex items-center justify-center rounded-lg dark:hover:bg-[#1F2937] hover:bg-gray-100 transition-colors"
        >
          {theme === 'dark' ? (
            <Sun size={16} className="text-gray-400" />
          ) : (
            <Moon size={16} className="text-gray-500" />
          )}
        </button>

        {/* Status pill */}
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full dark:bg-green-500/10 bg-green-50 dark:border border dark:border-green-500/20 border-green-200">
          <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
          <span className="text-[10px] font-medium text-green-500">All systems live</span>
        </div>
      </div>

      {pairScannerOpen && token && (
        <PairScannerModal token={token} onClose={() => setPairScannerOpen(false)} />
      )}
    </header>
  )
}
