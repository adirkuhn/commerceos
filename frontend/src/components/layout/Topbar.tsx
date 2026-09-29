import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Sun, Moon, Monitor, Command, QrCode } from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { PairScannerModal } from '../../scanner/PairScannerModal'
import NotificationBell from './NotificationBell'
import type { ThemePreference } from '../../theme/themePreference'
import { navTitle } from './navConfig'
import { LanguageSelect } from './LanguageSelect'

const APPEARANCE_ICON: Record<ThemePreference, React.ElementType> = { system: Monitor, light: Sun, dark: Moon }

export default function Topbar() {
  const { activeNav, themePreference, cycleThemePreference, setCommandOpen } = useApp()
  const { t } = useTranslation('layout')
  const appearanceLabel = { system: t('ui.appearanceSystem'), light: t('ui.appearanceLight'), dark: t('ui.appearanceDark') }[themePreference]
  const AppearanceIcon = APPEARANCE_ICON[themePreference]
  const { token } = useAuth()
  const [pairScannerOpen, setPairScannerOpen] = useState(false)

  return (
    <header className="h-14 shrink-0 flex items-center gap-4 px-5 border-b border-line bg-surface">
      <h1 className="text-sm font-semibold text-ink mr-2">
        {navTitle(activeNav)}
      </h1>

      {/* Search */}
      <button
        onClick={() => setCommandOpen(true)}
        className="flex-1 max-w-sm flex items-center gap-2 px-3 py-1.5 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-4 text-xs hover:bg-hover transition-colors"
      >
        <Search size={13} />
        <span className="flex-1 text-left">{t('ui.searchEverything')}</span>
        <span className="flex items-center gap-0.5 bg-surface px-1.5 py-0.5 rounded text-2xs text-ink-4 font-mono">
          <Command size={10} />K
        </span>
      </button>

      <div className="flex items-center gap-2 ml-auto">
        {/* Pair a handheld scanner -- global, not tied to any one page */}
        <button
          onClick={() => setPairScannerOpen(true)}
          title={t('ui.pairScanner')}
          aria-label={t('ui.pairScanner')}
          className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-hover transition-colors"
        >
          <QrCode size={16} className="text-ink-3" />
        </button>

        <LanguageSelect />

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
