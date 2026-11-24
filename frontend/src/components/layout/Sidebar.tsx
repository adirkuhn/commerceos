import { useEffect, useRef, useState } from 'react'
import { Check, ChevronDown, ChevronRight, LogOut } from 'lucide-react'
import { BrandMark } from '../brand/BrandMark'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { useProducts } from '../../pms/hooks/useProducts'
import type { NavItem } from '../../types'
import { appEnv } from '../../config/appEnv'
import { HOME_LINK, navGroupsForRole, roleLabel, sidebarActiveId, type NavGroup, type NavLink } from './navConfig'
import { useLook } from '../../theme/useLook'
import { LOOKS, type Look } from '../../theme/lookPreference'

// The rail: the look's rail colour in both appearances (navy for Ticket
// Wallet, graphite for Graphite). Its tokens (rail, rail-ink, ...) live in
// index.css. The current page is marked like a coupon's tab head: a filled
// row with a short rule in the mark colour before the label.

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
  const activeId = sidebarActiveId(activeNav)
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
          'relative w-full flex items-center gap-2.5 pl-4 pr-2.5 py-[5px] rounded-md text-left transition-colors duration-150 focus-visible:outline-rail-mark',
          active ? 'bg-rail-active text-rail-ink font-semibold' : 'text-rail-ink-2 hover:bg-rail-hover/10 hover:text-rail-ink',
        )}
      >
        <Icon size={16} className="shrink-0" aria-hidden />
        <span className="text-body flex-1 truncate">{link.label}</span>
        {active && <span className="absolute left-1 top-1/2 -translate-y-1/2 h-0.5 w-2 bg-rail-mark" aria-hidden />}
        {badge !== null && (
          <span className="text-2xs font-medium px-1.5 py-px rounded-sm bg-rail-hover/10 text-rail-ink tabular-nums">{badge}</span>
        )}
      </button>
    )
  }

  function renderGroup(group: NavGroup) {
    if (group.flat) return <div key={group.id} className="pt-3 space-y-px">{group.items.map(renderLink)}</div>
    const expanded = open.has(group.id)
    const listId = `nav-group-${group.id}`
    const Chevron = expanded ? ChevronDown : ChevronRight
    return (
      <div key={group.id} className="pt-4">
        <button
          onClick={() => toggle(group.id)}
          aria-expanded={expanded}
          aria-controls={listId}
          className="w-full flex items-center gap-1 pl-4 pr-3 py-1 rounded-md text-left caps-label-widest text-2xs font-semibold text-rail-ink-2 hover:text-rail-ink focus-visible:outline-rail-mark"
        >
          <span className="flex-1">{group.label}</span>
          <Chevron size={12} aria-hidden />
        </button>
        {expanded && (
          <div id={listId} className="mt-1 space-y-px">
            {group.items.map(renderLink)}
          </div>
        )}
      </div>
    )
  }

  return (
    <aside className="w-60 shrink-0 flex flex-col h-full bg-rail text-rail-ink">
      <div className="h-14 flex items-center gap-2.5 px-5 shrink-0">
        <BrandMark size={20} className="text-rail-ink" />
        <span className="font-bold text-sm tracking-tight text-rail-ink">CommerceOS</span>
      </div>

      <nav aria-label={t('ui.mainNav')} className="flex-1 min-h-0 flex flex-col">
        <div data-testid="nav-scroll" className="flex-1 overflow-y-auto py-3 px-2 space-y-0.5">
          {renderLink(HOME_LINK)}
          {groups.filter(g => !g.pinned).map(renderGroup)}
        </div>
        {groups.some(g => g.pinned) && (
          <div data-testid="nav-pinned" className="shrink-0 px-2 pb-2 max-h-[50%] overflow-y-auto border-t border-rail-hover/10">
            {groups.filter(g => g.pinned).map(renderGroup)}
          </div>
        )}
      </nav>

      <AccountMenu role={user?.role} />
    </aside>
  )
}

// deployedVersion is the release the server runs ("v1.0.11"), from
// /config.js -- empty in local dev.
function deployedVersion(): string {
  const v = appEnv().COMMERCEOS_VERSION?.trim()
  if (!v) return ''
  return v.startsWith('v') ? v : `v${v}`
}

const LOOK_LABEL_KEYS: Record<Look, 'ui.lookWallet' | 'ui.lookGraphite'> = {
  wallet: 'ui.lookWallet',
  graphite: 'ui.lookGraphite',
}

function AccountMenu({ role }: { role?: string }) {
  const { logout } = useAuth()
  const { look, setLook } = useLook()
  const { t } = useTranslation('layout')
  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const itemRef = useRef<HTMLButtonElement>(null)
  const label = roleLabel(role)
  const version = deployedVersion()

  useEffect(() => {
    if (open) itemRef.current?.focus()
  }, [open])

  function close() {
    setOpen(false)
    buttonRef.current?.focus()
  }

  return (
    <div className="relative p-3 shrink-0 border-t border-rail-hover/10">
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div
            role="menu"
            aria-label={t('ui.account')}
            onKeyDown={e => { if (e.key === 'Escape') close() }}
            className="absolute bottom-full left-3 right-3 mb-2 z-50 rounded-lg bg-raised border border-line shadow-popover p-1"
          >
            <p className="px-2.5 pt-1.5 pb-1 text-2xs text-ink-4">{t('ui.signedInAs')}</p>
            <p className="px-2.5 pb-2 text-xs font-medium text-ink">{label}</p>
            <div className="border-t border-line my-1" />
            <p className="px-2.5 pt-1 pb-1 caps-label-widest text-2xs font-semibold text-ink-4">{t('ui.look')}</p>
            {LOOKS.map((option, i) => (
              <button
                key={option}
                ref={i === 0 ? itemRef : undefined}
                role="menuitemradio"
                aria-checked={look === option}
                onClick={() => { setLook(option); setOpen(false) }}
                className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md text-left text-xs text-ink-2 hover:bg-hover"
              >
                <Check size={14} aria-hidden className={look === option ? 'text-accent' : 'invisible'} />
                {t(LOOK_LABEL_KEYS[option])}
              </button>
            ))}
            <div className="border-t border-line my-1" />
            <button
              role="menuitem"
              onClick={logout}
              className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md text-left text-xs text-ink-2 hover:bg-hover"
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
        className="w-full flex items-center gap-2.5 px-2 py-1.5 rounded-md text-left hover:bg-rail-hover/10 focus-visible:outline-rail-mark"
      >
        <span className="w-7 h-7 rounded-full flex items-center justify-center text-2xs font-semibold shrink-0 bg-rail-hover/10 text-rail-ink" aria-hidden>
          {label.charAt(0)}
        </span>
        <span className="flex-1 min-w-0 flex flex-col">
          <span className="text-xs font-medium truncate text-rail-ink">{label}</span>
          {version && <span className="text-2xs text-rail-ink-2 truncate tabular-nums">{version}</span>}
        </span>
        <ChevronDown size={12} className="shrink-0 text-rail-ink-2" aria-hidden />
      </button>
    </div>
  )
}
