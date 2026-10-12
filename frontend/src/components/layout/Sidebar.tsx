import { useEffect, useRef, useState } from 'react'
import { ChevronDown, ChevronRight, LogOut, Palette } from 'lucide-react'
import { BrandMark } from '../brand/BrandMark'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { useProducts } from '../../pms/hooks/useProducts'
import type { NavItem } from '../../types'
import { homeLink, navGroupsForRole, roleLabel, sidebarActiveId, type NavGroup, type NavLink } from './navConfig'
import { useDesign } from '../../theme/useDesign'
import type { Design } from '../../theme/designPreference'

// Class sets per design -- classic is the pre-2026-10 sidebar, unchanged;
// depot is the dark ultramarine sidebar (dark in both appearances).
interface SidebarStyle {
  aside: string
  brandRow: string
  brandName: string
  link: string
  linkActive: string
  linkIdle: string
  badge: string
  groupHeader: string
  accountRow: string
  accountButton: string
  accountName: string
  avatar: string
  chevron: string
}

const STYLES: Record<Design, SidebarStyle> = {
  classic: {
    aside: 'bg-surface border-r border-line',
    brandRow: 'border-b border-line',
    brandName: 'text-ink',
    link: 'rounded-lg',
    linkActive: 'bg-accent/15 text-accent',
    linkIdle: 'text-ink-3 hover:bg-hover hover:text-ink',
    badge: 'bg-fill text-ink-3',
    groupHeader: 'caps-label text-2xs text-ink-4 hover:text-ink-2',
    accountRow: 'border-t border-line',
    accountButton: 'hover:bg-hover',
    accountName: 'text-ink',
    avatar: 'bg-fill border border-line text-ink-2',
    chevron: 'text-ink-4',
  },
  depot: {
    aside: 'bg-depot text-depot-ink',
    brandRow: '',
    brandName: 'text-depot-ink',
    link: 'rounded-lg depot:py-[3px] focus-visible:outline-depot-mark',
    linkActive: 'bg-accent-solid text-white shadow-[inset_3px_0_0_rgb(var(--depot-mark))]',
    linkIdle: 'text-depot-ink-2 hover:bg-depot-hover/10 hover:text-depot-ink',
    badge: 'bg-depot-hover/10 text-depot-ink',
    groupHeader: 'text-xs text-depot-ink-2 hover:text-depot-ink depot:py-0.5 focus-visible:outline-depot-mark',
    accountRow: 'border-t border-depot-hover/10',
    accountButton: 'hover:bg-depot-hover/10 focus-visible:outline-depot-mark',
    accountName: 'text-depot-ink',
    avatar: 'bg-depot-hover/10 text-depot-ink',
    chevron: 'text-depot-ink-2',
  },
}

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

  const { design } = useDesign()
  const style = STYLES[design]
  const groups = navGroupsForRole(user?.role, design)
  const activeId = sidebarActiveId(activeNav, design)
  const [open, setOpen] = useState(() => initiallyOpen(groups, activeId))

  // Navigating into a closed group (URL, search, Back) opens it, so the
  // current page is always visible in the sidebar.
  const activeGroup = groups.find(g => g.items.some(i => i.id === activeId))?.id
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
    const active = activeId === link.id
    const Icon = link.icon
    const badge = link.id === 'products' && total > 0 ? total : null
    return (
      <button
        key={link.id}
        onClick={() => setActiveNav(link.id)}
        aria-current={active ? 'page' : undefined}
        className={clsx(
          'w-full flex items-center gap-2.5 px-2.5 py-1.5 text-left transition-colors',
          style.link,
          active ? style.linkActive : style.linkIdle,
        )}
      >
        <Icon size={16} className="shrink-0" aria-hidden />
        <span className="text-body font-medium flex-1 truncate">{link.label}</span>
        {badge !== null && (
          <span className={clsx('text-2xs font-medium px-1.5 py-0.5 rounded-full tabular-nums', style.badge)}>{badge}</span>
        )}
      </button>
    )
  }

  return (
    <aside className={clsx('w-60 shrink-0 flex flex-col h-full', style.aside)}>
      <div className={clsx('h-14 flex items-center gap-2 px-4 shrink-0', style.brandRow)}>
        <div className="w-7 h-7 rounded-lg bg-accent-solid flex items-center justify-center">
          <BrandMark size={18} className="text-white" />
        </div>
        <span className={clsx('font-semibold text-sm tracking-tight', style.brandName)}>CommerceOS</span>
      </div>

      <nav aria-label={t('ui.mainNav')} className="flex-1 flex flex-col overflow-y-auto py-3 px-2 space-y-0.5">
        {renderLink(homeLink(design))}
        {groups.map(group => {
          if (group.flat) return <div key={group.id} className="pt-3 depot:pt-2 space-y-0.5">{group.items.map(renderLink)}</div>
          const expanded = open.has(group.id)
          const listId = `nav-group-${group.id}`
          const Chevron = expanded ? ChevronDown : ChevronRight
          return (
            <div key={group.id} className={clsx('pt-3 depot:pt-2', design === 'depot' && group.id === 'setup' && 'mt-auto')}>
              <button
                onClick={() => toggle(group.id)}
                aria-expanded={expanded}
                aria-controls={listId}
                className={clsx('w-full flex items-center gap-1 px-2.5 py-1 rounded-md text-left font-semibold', style.groupHeader)}
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

      <AccountMenu role={user?.role} style={style} />
    </aside>
  )
}

function AccountMenu({ role, style }: { role?: string; style: SidebarStyle }) {
  const { logout } = useAuth()
  const { design, setDesign } = useDesign()
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
    <div className={clsx('relative p-3 shrink-0', style.accountRow)}>
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
            {/* The redesign stays switchable: back to classic (or forward
                again) without a release. */}
            <button
              ref={itemRef}
              role="menuitem"
              onClick={() => { setDesign(design === 'depot' ? 'classic' : 'depot'); setOpen(false) }}
              className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-left text-xs text-ink-2 hover:bg-hover"
            >
              <Palette size={14} aria-hidden />
              {design === 'depot' ? t('ui.useClassicDesign') : t('ui.useNewDesign')}
            </button>
            <button
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
        className={clsx('w-full flex items-center gap-2.5 px-2 py-1.5 rounded-lg text-left', style.accountButton)}
      >
        <span className={clsx('w-7 h-7 rounded-full flex items-center justify-center text-2xs font-semibold shrink-0', style.avatar)} aria-hidden>
          {label.charAt(0)}
        </span>
        <span className={clsx('flex-1 min-w-0 text-xs font-medium truncate', style.accountName)}>{label}</span>
        <ChevronDown size={12} className={clsx('shrink-0', style.chevron)} aria-hidden />
      </button>
    </div>
  )
}
