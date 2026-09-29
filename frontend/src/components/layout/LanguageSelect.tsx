import { Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { setLanguage } from '../../i18n/i18n'
import { LOCALES, LOCALE_NAMES, resolveLocale, type Locale } from '../../i18n/locale'

/** The admin language for this user (remembered in this browser). */
export function LanguageSelect() {
  const { t, i18n } = useTranslation('layout')
  return (
    <label className="flex items-center gap-1 text-ink-3" title={t('ui.language')}>
      <Languages size={15} aria-hidden />
      <span className="sr-only">{t('ui.language')}</span>
      <select
        value={resolveLocale(i18n.language)}
        onChange={e => void setLanguage(e.target.value as Locale)}
        className="bg-transparent text-xs text-ink-2 outline-none cursor-pointer"
      >
        {LOCALES.map(l => <option key={l} value={l}>{LOCALE_NAMES[l]}</option>)}
      </select>
    </label>
  )
}
