import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search, LayoutDashboard, Package, Warehouse, ShoppingCart, ScanBarcode,
  Settings, Factory, ShoppingBag, Plug, Truck, BarChart3, Users, Loader2,
} from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { navItemToPath } from '../../routes'
import { globalSearch } from '../../search/api'
import type { GlobalSearchResult } from '../../search/types'
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
  { label: 'Go to Reports', icon: BarChart3, nav: 'reports-overview' },
  { label: 'Go to Users & Roles', icon: Users, nav: 'users' },
  { label: 'Go to Settings', icon: Settings, nav: 'settings' },
]

// MIN_QUERY_LENGTH_FOR_SEARCH: a single character against every entity
// table is both slow and useless (hundreds of near-random matches) --
// require at least this many before firing the backend call. Navigation
// commands are unaffected, those stay instant/client-side at any length.
const MIN_QUERY_LENGTH_FOR_SEARCH = 2
const SEARCH_DEBOUNCE_MS = 250

// One flat, keyboard-navigable list entry -- Navigation commands and each
// entity search result are normalized to this shape so ArrowUp/ArrowDown/
// Enter can move through every section as a single sequence, the way a
// real Cmd+K palette behaves (as opposed to per-section, independent lists).
interface PaletteEntry {
  key: string
  section: string
  icon: React.ElementType
  label: string
  sublabel?: string
  shortcut?: string
  onSelect: () => void
}

export default function CommandPalette() {
  const { commandOpen, setCommandOpen, setActiveNav } = useApp()
  const { token } = useAuth()
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const [results, setResults] = useState<GlobalSearchResult | null>(null)
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState('')

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
      setResults(null)
      setSearchError('')
      setActiveIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [commandOpen])

  // Debounced backend search -- Navigation commands filter instantly below,
  // this only covers the cross-entity part (Products/Suppliers/Customers/
  // Sales Orders).
  useEffect(() => {
    const trimmed = query.trim()
    if (trimmed.length < MIN_QUERY_LENGTH_FOR_SEARCH || !token) {
      setResults(null)
      setSearching(false)
      setSearchError('')
      return
    }
    setSearching(true)
    setSearchError('')
    const timer = setTimeout(() => {
      globalSearch(token, trimmed)
        .then(setResults)
        .catch(err => setSearchError(err instanceof Error ? err.message : 'Search failed'))
        .finally(() => setSearching(false))
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [query, token])

  if (!commandOpen) return null

  function openNav(nav: NavItem) {
    setActiveNav(nav)
    setCommandOpen(false)
  }

  function openRecord(nav: NavItem, id: string) {
    navigate(`${navItemToPath(nav)}?id=${id}`)
    setCommandOpen(false)
  }

  const navEntries: PaletteEntry[] = COMMANDS
    .filter(c => query.trim() === '' || c.label.toLowerCase().includes(query.trim().toLowerCase()))
    .map(cmd => ({
      key: `nav-${cmd.nav}`, section: 'Navigation', icon: cmd.icon, label: cmd.label,
      shortcut: cmd.shortcut, onSelect: () => openNav(cmd.nav),
    }))

  const productEntries: PaletteEntry[] = (results?.products ?? []).map(p => ({
    key: `product-${p.id}`, section: 'Products', icon: Package, label: p.name,
    onSelect: () => openRecord('products', p.id),
  }))
  const supplierEntries: PaletteEntry[] = (results?.suppliers ?? []).map(s => ({
    key: `supplier-${s.id}`, section: 'Suppliers', icon: Factory, label: s.name, sublabel: s.email,
    onSelect: () => openRecord('suppliers', s.id),
  }))
  const customerEntries: PaletteEntry[] = (results?.customers ?? []).map(c => ({
    key: `customer-${c.id}`, section: 'Customers', icon: Users,
    label: [c.first_name, c.last_name].filter(Boolean).join(' ') || c.email || 'Customer',
    sublabel: c.email, onSelect: () => openRecord('customers', c.id),
  }))
  const salesOrderEntries: PaletteEntry[] = (results?.sales_orders ?? []).map(o => ({
    key: `order-${o.id}`, section: 'Sales Orders', icon: ShoppingCart, label: `#${o.number}`,
    sublabel: [o.billing_first_name, o.billing_last_name].filter(Boolean).join(' ') || o.status,
    onSelect: () => openRecord('sales-orders', o.id),
  }))

  const sections: { title: string; entries: PaletteEntry[] }[] = [
    { title: 'Navigation', entries: navEntries },
    { title: 'Products', entries: productEntries },
    { title: 'Suppliers', entries: supplierEntries },
    { title: 'Customers', entries: customerEntries },
    { title: 'Sales Orders', entries: salesOrderEntries },
  ].filter(s => s.entries.length > 0)

  const flatEntries = sections.flatMap(s => s.entries)
  const hasBackendSearch = query.trim().length >= MIN_QUERY_LENGTH_FOR_SEARCH

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex(i => Math.min(i + 1, Math.max(flatEntries.length - 1, 0)))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex(i => Math.max(i - 1, 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      flatEntries[activeIndex]?.onSelect()
    }
  }

  let renderedIndex = -1

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      <div
        className="absolute inset-0 dark:bg-black/60 bg-black/30 backdrop-blur-sm"
        onClick={() => setCommandOpen(false)}
      />
      <div className="relative w-full max-w-lg dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="flex items-center gap-3 px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          {searching ? (
            <Loader2 size={16} className="dark:text-gray-400 text-gray-400 shrink-0 animate-spin" />
          ) : (
            <Search size={16} className="dark:text-gray-400 text-gray-400 shrink-0" />
          )}
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={e => { setQuery(e.target.value); setActiveIndex(0) }}
            onKeyDown={handleKeyDown}
            placeholder="Search everything..."
            className="flex-1 bg-transparent text-sm dark:text-gray-100 text-gray-900 dark:placeholder-gray-500 placeholder-gray-400 outline-none"
          />
          <kbd className="text-[10px] dark:bg-[#111827] bg-gray-100 dark:text-gray-500 text-gray-400 px-1.5 py-1 rounded font-mono">ESC</kbd>
        </div>
        <div className="py-2 max-h-96 overflow-y-auto">
          {searchError && (
            <p className="px-4 py-2 text-xs text-red-400">{searchError}</p>
          )}
          {flatEntries.length === 0 && !searching && (
            <p className="px-4 py-4 text-sm dark:text-gray-500 text-gray-400 text-center">
              {hasBackendSearch ? 'No results' : 'No matching commands'}
            </p>
          )}
          {sections.map(section => (
            <div key={section.title}>
              <p className="px-4 py-1.5 text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">
                {section.title}
              </p>
              {section.entries.map(entry => {
                renderedIndex += 1
                const isActive = renderedIndex === activeIndex
                const Icon = entry.icon
                return (
                  <button
                    key={entry.key}
                    onClick={entry.onSelect}
                    onMouseEnter={() => setActiveIndex(renderedIndex)}
                    className={`w-full flex items-center gap-3 px-4 py-2.5 transition-colors ${
                      isActive ? 'dark:bg-[#2D3748] bg-gray-50' : 'hover:dark:bg-[#2D3748] hover:bg-gray-50'
                    }`}
                  >
                    <Icon size={15} className="dark:text-gray-400 text-gray-500 shrink-0" />
                    <span className="flex-1 min-w-0 text-left">
                      <span className="block text-sm dark:text-gray-200 text-gray-700 truncate">{entry.label}</span>
                      {entry.sublabel && (
                        <span className="block text-[11px] dark:text-gray-500 text-gray-400 truncate">{entry.sublabel}</span>
                      )}
                    </span>
                    {entry.shortcut && (
                      <span className="text-[10px] dark:text-gray-600 text-gray-400 font-mono shrink-0">{entry.shortcut}</span>
                    )}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
