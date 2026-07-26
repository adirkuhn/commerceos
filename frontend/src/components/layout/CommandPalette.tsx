import { useEffect, useRef, useState } from 'react'
import {
  Search, LayoutDashboard, Package, Warehouse, ShoppingCart, ScanBarcode,
  Settings, Factory, ShoppingBag, Plug, Truck, BarChart3, Users,
} from 'lucide-react'
import { useApp } from '../../context/AppContext'
import type { NavItem } from '../../types'

const COMMANDS: { label: string; icon: React.ElementType; nav: NavItem; shortcut?: string }[] = [
  { label: 'Go to Dashboard', icon: LayoutDashboard, nav: 'dashboard', shortcut: 'G D' },
  { label: 'Go to Products', icon: Package, nav: 'products', shortcut: 'G P' },
  { label: 'Go to Inventory', icon: Warehouse, nav: 'inventory', shortcut: 'G I' },
  { label: 'Go to Orders', icon: ShoppingCart, nav: 'orders', shortcut: 'G O' },
  { label: 'Go to Picking & Packing', icon: ScanBarcode, nav: 'picking', shortcut: 'G W' },
  { label: 'Go to Suppliers', icon: Factory, nav: 'suppliers' },
  { label: 'Go to Procurement', icon: ShoppingBag, nav: 'procurement' },
  { label: 'Go to Integrations', icon: Plug, nav: 'integrations' },
  { label: 'Go to Shipping', icon: Truck, nav: 'shipping' },
  { label: 'Go to Reports', icon: BarChart3, nav: 'reports' },
  { label: 'Go to Users & Roles', icon: Users, nav: 'users' },
  { label: 'Go to Settings', icon: Settings, nav: 'settings' },
]

export default function CommandPalette() {
  const { commandOpen, setCommandOpen, setActiveNav } = useApp()
  const inputRef = useRef<HTMLInputElement>(null)
  const [query, setQuery] = useState('')

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setCommandOpen(true)
      }
      if (e.key === 'Escape') setCommandOpen(false)
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [setCommandOpen])

  useEffect(() => {
    if (commandOpen) {
      setQuery('')
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [commandOpen])

  if (!commandOpen) return null

  const filtered = COMMANDS.filter(c =>
    query === '' || c.label.toLowerCase().includes(query.toLowerCase())
  )

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      <div
        className="absolute inset-0 dark:bg-black/60 bg-black/30 backdrop-blur-sm"
        onClick={() => setCommandOpen(false)}
      />
      <div className="relative w-full max-w-lg dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="flex items-center gap-3 px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <Search size={16} className="dark:text-gray-400 text-gray-400 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search commands, navigate..."
            className="flex-1 bg-transparent text-sm dark:text-gray-100 text-gray-900 dark:placeholder-gray-500 placeholder-gray-400 outline-none"
          />
          <kbd className="text-[10px] dark:bg-[#111827] bg-gray-100 dark:text-gray-500 text-gray-400 px-1.5 py-1 rounded font-mono">ESC</kbd>
        </div>
        <div className="py-2 max-h-80 overflow-y-auto">
          <p className="px-4 py-1.5 text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Navigation</p>
          {filtered.length === 0 && (
            <p className="px-4 py-4 text-sm dark:text-gray-500 text-gray-400 text-center">No results</p>
          )}
          {filtered.map(cmd => {
            const Icon = cmd.icon
            return (
              <button
                key={cmd.nav}
                onClick={() => { setActiveNav(cmd.nav); setCommandOpen(false) }}
                className="w-full flex items-center gap-3 px-4 py-2.5 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors"
              >
                <Icon size={15} className="dark:text-gray-400 text-gray-500 shrink-0" />
                <span className="flex-1 text-sm dark:text-gray-200 text-gray-700 text-left">{cmd.label}</span>
                {cmd.shortcut && (
                  <span className="text-[10px] dark:text-gray-600 text-gray-400 font-mono">{cmd.shortcut}</span>
                )}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
