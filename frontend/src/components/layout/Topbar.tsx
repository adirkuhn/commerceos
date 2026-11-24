import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Sun, Moon, Monitor, Command, QrCode, Eye, EyeOff, Menu } from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { PairScannerModal } from '../../scanner/PairScannerModal'
import NotificationBell from './NotificationBell'
import type { ThemePreference } from '../../theme/themePreference'
import { navTitle } from './navConfig'
import { LanguageSelect } from './LanguageSelect'
import { useAmountVisibility } from '../../format/amountVisibility'

const APPEARANCE_ICON: Record<ThemePreference, React.ElementType> = { system: Monitor, light: Sun, dark: Moon }

export default function Topbar() {
  const { activeNav, themePreference, cycleThemePreference, setCommandOpen, navOpen, setNavOpen } = useApp()
  const { t } = useTranslation('layout')
  const appearanceLabel = { system: t('ui.appearanceSystem'), light: t('ui.appearanceLight'), dark: t('ui.appearanceDark') }[themePreference]
  const AppearanceIcon = APPEARANCE_ICON[themePreference]
  const { token } = useAuth()
  const [pairScannerOpen, setPairScannerOpen] = useState(false)
  const amounts = useAmountVisibility()
  const amountsLabel = amounts.visible ? t('ui.hideAmounts') : t('ui.showAmounts')
  const AmountsIcon = amounts.visible ? Eye : EyeOff

  return (
    <header className="h-14 shrink-0 flex items-center gap-2 sm:gap-4 px-3 sm:px-5 border-b border-line bg-surface">
      {/* Small screens: the sidebar is a drawer (see Sidebar). */}
      <button
        onClick={() => setNavOpen(!navOpen)}
        aria-label={t('ui.menu')}
        aria-expanded={navOpen}
        aria-controls="main-sidebar"
        className="lg:hidden w-9 h-9 -ml-1 flex items-center justify-center rounded-md hover:bg-hover transition-colors"
      >
        <Menu size={18} className="text-ink-2" />
      </button>

      <h1 className="min-w-0 truncate text-sm font-bold text-ink sm:mr-2">
        {navTitle(activeNav)}
      </h1>

      {/* Search */}
      <button
        onClick={() => setCommandOpen(true)}
        aria-label={t('ui.searchEverything')}
        className="shrink-0 sm:shrink sm:flex-1 max-w-sm flex items-center gap-2 px-2.5 sm:px-3 py-1.5 rounded-md border border-line bg-sunken text-ink-4 text-xs hover:bg-hover hover:text-ink-3 transition-colors"
      >
        <Search size={14} />
        <span className="hidden sm:inline flex-1 text-left">{t('ui.searchEverything')}</span>
        <span className="hidden sm:flex items-center gap-0.5 border border-line bg-surface px-1.5 py-0.5 rounded text-2xs text-ink-4 font-mono">
          <Command size={10} />K
        </span>
      </button>

      <div className="flex items-center gap-1 sm:gap-2 ml-auto">
        {/* Money privacy: amounts start hidden; this global choice is remembered */}
        <button
          onClick={() => amounts.setVisible(!amounts.visible)}
          aria-pressed={amounts.visible}
          aria-label={amountsLabel}
          title={amountsLabel}
          className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-hover transition-colors"
        >
          <AmountsIcon size={16} className="text-ink-3" />
        </button>

        {/* Pair a handheld scanner -- global, not tied to any one page */}
        <button
          onClick={() => setPairScannerOpen(true)}
          title={t('ui.pairScanner')}
          aria-label={t('ui.pairScanner')}
          className="hidden sm:flex w-8 h-8 items-center justify-center rounded-lg hover:bg-hover transition-colors"
        >
          <QrCode size={16} className="text-ink-3" />
        </button>

        <div className="hidden md:block"><LanguageSelect /></div>

        {/* Notifications */}
        <NotificationBell />

        {/* Appearance: follows the OS by default, click to override */}
        <button
          onClick={cycleThemePreference}
          aria-label={t('ui.appearance', { mode: appearanceLabel })}
          title={t('ui.appearance', { mode: appearanceLabel })}
          className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-hover transition-colors"
        >
          <AppearanceIcon size={16} className="text-ink-3" />
        </button>
      </div>

      {pairScannerOpen && token && (
        <PairScannerModal token={token} onClose={() => setPairScannerOpen(false)} />
      )}
    </header>
  )
}
