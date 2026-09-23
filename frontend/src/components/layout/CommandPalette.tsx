import { useEffect, useId, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, Package, ShoppingCart, Factory, Users, Loader2 } from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { navItemToPath } from '../../routes'
import { globalSearch } from '../../search/api'
import { Dialog } from '../common/Dialog'
import type { GlobalSearchResult } from '../../search/types'
import type { NavItem } from '../../types'
import { HOME_LINK, navGroupsForRole, navTitle, shortcutFor } from './navConfig'

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
  const { token, user } = useAuth()
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const listboxId = useId()
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

  // Keep the keyboard-highlighted row visible in the scrolling list.
  useEffect(() => {
    if (!commandOpen) return
    document.getElementById(`${listboxId}-option-${activeIndex}`)?.scrollIntoView?.({ block: 'nearest' })
  }, [activeIndex, commandOpen, listboxId])

  if (!commandOpen) return null

  function openNav(nav: NavItem) {
    setActiveNav(nav)
    setCommandOpen(false)
  }

  function openRecord(nav: NavItem, id: string) {
    navigate(`${navItemToPath(nav)}/${id}`)
    setCommandOpen(false)
  }

  // Same pages, names and role gating as the sidebar.
  const navEntries: PaletteEntry[] = [HOME_LINK, ...navGroupsForRole(user?.role).flatMap(g => g.items)]
    .map(link => ({ link, label: `Go to ${navTitle(link.id)}` }))
    .filter(({ label }) => query.trim() === '' || label.toLowerCase().includes(query.trim().toLowerCase()))
    .map(({ link, label }) => ({
      key: `nav-${link.id}`, section: 'Navigation', icon: link.icon, label,
      shortcut: shortcutFor(link.id), onSelect: () => openNav(link.id),
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
  const optionId = (index: number) => `${listboxId}-option-${index}`
  const activeEntryExists = activeIndex < flatEntries.length

  return (
    <Dialog onClose={() => setCommandOpen(false)} label="Search" className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      <div
        className="absolute inset-0 dark:bg-black/60 bg-black/30 backdrop-blur-sm"
        onClick={() => setCommandOpen(false)}
      />
      <div className="relative w-full max-w-lg bg-raised rounded-xl shadow-2xl border border-line overflow-hidden">
        <div className="flex items-center gap-3 px-4 py-3 border-b border-line">
          {searching ? (
            <Loader2 size={16} className="text-ink-4 shrink-0 animate-spin" />
          ) : (
            <Search size={16} className="text-ink-4 shrink-0" />
          )}
          <input
            ref={inputRef}
            type="text"
            role="combobox"
            aria-label="Search"
            aria-expanded={flatEntries.length > 0}
            aria-controls={listboxId}
            aria-autocomplete="list"
            aria-activedescendant={activeEntryExists ? optionId(activeIndex) : undefined}
            value={query}
            onChange={e => { setQuery(e.target.value); setActiveIndex(0) }}
            onKeyDown={handleKeyDown}
            placeholder="Search everything..."
            className="flex-1 bg-transparent text-sm text-ink placeholder-ink-4 outline-none"
          />
          <kbd className="text-2xs bg-fill text-ink-4 px-1.5 py-1 rounded font-mono">ESC</kbd>
        </div>
        <div id={listboxId} role="listbox" aria-label="Results" className="py-2 max-h-96 overflow-y-auto">
          {searchError && (
            <p role="alert" className="px-4 py-2 text-xs text-danger">{searchError}</p>
          )}
          {flatEntries.length === 0 && !searching && (
            <p className="px-4 py-4 text-sm text-ink-4 text-center">
              {hasBackendSearch ? 'No results' : 'No matching commands'}
            </p>
          )}
          {sections.map(section => (
            <div key={section.title} role="group" aria-labelledby={`${listboxId}-${section.title}`}>
              <p id={`${listboxId}-${section.title}`} className="px-4 py-1.5 text-2xs font-medium text-ink-4 uppercase tracking-wider">
                {section.title}
              </p>
              {section.entries.map(entry => {
                renderedIndex += 1
                // Captured per row: the closure must not see the counter's
                // final value (hovering any row used to highlight the last).
                const index = renderedIndex
                const isActive = index === activeIndex
                const Icon = entry.icon
                return (
                  <button
                    key={entry.key}
                    id={optionId(index)}
                    role="option"
                    aria-selected={isActive}
                    tabIndex={-1}
                    onClick={entry.onSelect}
                    onMouseEnter={() => setActiveIndex(index)}
                    className={`w-full flex items-center gap-3 px-4 py-2.5 transition-colors ${
                      isActive ? 'bg-hover' : 'hover:bg-hover'
                    }`}
                  >
                    <Icon size={15} className="text-ink-3 shrink-0" />
                    <span className="flex-1 min-w-0 text-left">
                      <span className="block text-sm text-ink-2 truncate">{entry.label}</span>
                      {entry.sublabel && (
                        <span className="block text-2xs text-ink-4 truncate">{entry.sublabel}</span>
                      )}
                    </span>
                    {entry.shortcut && (
                      <kbd className="text-2xs text-ink-4 font-mono shrink-0">{entry.shortcut}</kbd>
                    )}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </div>
    </Dialog>
  )
}
