import { useEffect, useRef, useState } from 'react'
import { ChevronDown, ChevronRight, LogOut } from 'lucide-react'
import { BrandMark } from '../brand/BrandMark'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { useProducts } from '../../pms/hooks/useProducts'
import type { NavItem } from '../../types'
import { HOME_LINK, navGroupsForRole, roleLabel, type NavGroup, type NavLink } from './navConfig'

function initiallyOpen(groups: NavGroup[], active: NavItem): Set<string> {
  return new Set(groups
    .filter(g => !g.collapsedByDefault || g.items.some(i => i.id === active))
    .map(g => g.id))
}

export default function Sidebar() {
  const { activeNav, setActiveNav } = useApp()
  // Re-renders the menu when the language changes (labels come from i18next).
  const { t } = useTranslation('layout')
  const { user, token } = useAuth()
  // Only page 1 is fetched here — the badge needs the server-side total,
  // not the current page's item count, now that products are paginated.
  const { total } = useProducts(token ?? '', { page: 1, perPage: 1 })

  const groups = navGroupsForRole(user?.role)
  const [open, setOpen] = useState(() => initiallyOpen(groups, activeNav))

  // Navigating into a closed group (URL, search, Back) opens it, so the
  // current page is always visible in the sidebar.
  const activeGroup = groups.find(g => g.items.some(i => i.id === activeNav))?.id
  useEffect(() => {
    if (activeGroup) setOpen(prev => (prev.has(activeGroup) ? prev : new Set(prev).add(activeGroup)))
  }, [activeGroup])

  function toggle(id: string) {
    setOpen(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function renderLink(link: NavLink) {
    const active = activeNav === link.id
    const Icon = link.icon
    const badge = link.id === 'products' && total > 0 ? total : null
    return (
      <button
        key={link.id}
        onClick={() => setActiveNav(link.id)}
        aria-current={active ? 'page' : undefined}
        className={clsx(
          'w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-left transition-colors',
          active ? 'bg-accent/15 text-accent' : 'text-ink-3 hover:bg-hover hover:text-ink',
        )}
      >
        <Icon size={16} className="shrink-0" aria-hidden />
        <span className="text-body font-medium flex-1 truncate">{link.label}</span>
        {badge !== null && (
          <span className="text-2xs font-medium px-1.5 py-0.5 rounded-full bg-fill text-ink-3 tabular-nums">{badge}</span>
        )}
      </button>
    )
  }

  return (
    <aside className="w-60 shrink-0 flex flex-col h-full bg-surface border-r border-line">
      <div className="h-14 flex items-center gap-2 px-4 border-b border-line shrink-0">
        <div className="w-7 h-7 rounded-lg bg-accent-solid flex items-center justify-center">
          <BrandMark size={18} className="text-white" />
        </div>
        <span className="font-semibold text-sm text-ink tracking-tight">CommerceOS</span>
      </div>

      <nav aria-label={t('ui.mainNav')} className="flex-1 overflow-y-auto py-3 px-2 space-y-0.5">
        {renderLink(HOME_LINK)}
        {groups.map(group => {
          const expanded = open.has(group.id)
          const listId = `nav-group-${group.id}`
          const Chevron = expanded ? ChevronDown : ChevronRight
          return (
            <div key={group.id} className="pt-3">
              <button
                onClick={() => toggle(group.id)}
                aria-expanded={expanded}
                aria-controls={listId}
                className="w-full flex items-center gap-1 px-2.5 py-1 rounded-md text-left text-2xs font-semibold caps-label text-ink-4 hover:text-ink-2"
              >
                <span className="flex-1">{group.label}</span>
                <Chevron size={12} aria-hidden />
              </button>
              {expanded && (
                <div id={listId} className="mt-0.5 space-y-0.5">
                  {group.items.map(renderLink)}
                </div>
              )}
            </div>
          )
        })}
      </nav>

      <AccountMenu role={user?.role} />
    </aside>
  )
}

function AccountMenu({ role }: { role?: string }) {
  const { logout } = useAuth()
  const { t } = useTranslation('layout')
  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const itemRef = useRef<HTMLButtonElement>(null)
  const label = roleLabel(role)

  useEffect(() => {
    if (open) itemRef.current?.focus()
  }, [open])

  function close() {
    setOpen(false)
    buttonRef.current?.focus()
  }

  return (
    <div className="relative p-3 border-t border-line shrink-0">
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div
            role="menu"
            aria-label={t('ui.account')}
            onKeyDown={e => { if (e.key === 'Escape') close() }}
            className="absolute bottom-full left-3 right-3 mb-2 z-50 rounded-xl bg-raised border border-line shadow-2xl p-1"
          >
            <p className="px-2.5 pt-1.5 pb-1 text-2xs text-ink-4">{t('ui.signedInAs')}</p>
            <p className="px-2.5 pb-2 text-xs font-medium text-ink">{label}</p>
            <div className="border-t border-line my-1" />
            <button
              ref={itemRef}
              role="menuitem"
              onClick={logout}
              className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-left text-xs text-ink-2 hover:bg-hover"
            >
              <LogOut size={14} aria-hidden />
              {t('ui.signOut')}
            </button>
          </div>
        </>
      )}
      <button
        ref={buttonRef}
        onClick={() => setOpen(o => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t('ui.accountOf', { name: label })}
        className="w-full flex items-center gap-2.5 px-2 py-1.5 rounded-lg hover:bg-hover text-left"
      >
        <span className="w-7 h-7 rounded-full bg-fill border border-line flex items-center justify-center text-2xs font-semibold text-ink-2 shrink-0" aria-hidden>
          {label.charAt(0)}
        </span>
        <span className="flex-1 min-w-0 text-xs font-medium text-ink truncate">{label}</span>
        <ChevronDown size={12} className="text-ink-4 shrink-0" aria-hidden />
      </button>
    </div>
  )
}
