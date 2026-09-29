import { useTranslation } from 'react-i18next'
import type { StorefrontLocale } from '../types'
import { LOCALES, LOCALE_NAMES } from '../../i18n/locale'

interface Props {
  // "" = the company language (Settings → Localization); only a new channel
  // starts that way.
  locale: StorefrontLocale | ''
  onLocaleChange: (locale: StorefrontLocale | '') => void
  // Other languages shoppers can switch to (never the default itself).
  others: StorefrontLocale[]
  onOthersChange: (others: StorefrontLocale[]) => void
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const labelClass = 'block text-2xs text-ink-4 uppercase tracking-wider mb-1.5'

export default function StorefrontLanguagesField({ locale, onLocaleChange, others, onOthersChange }: Props) {
  const { t } = useTranslation('channels')

  function changeDefault(next: StorefrontLocale | '') {
    onLocaleChange(next)
    if (others.some(l => l === next)) onOthersChange(others.filter(l => l !== next))
  }

  function toggle(l: StorefrontLocale, on: boolean) {
    onOthersChange(on ? [...others, l] : others.filter(o => o !== l))
  }

  return (
    <>
      <div>
        <label htmlFor="channel-storefront-locale" className={labelClass}>{t('storefront.language')}</label>
        <select id="channel-storefront-locale" value={locale} onChange={e => changeDefault(e.target.value as StorefrontLocale | '')} className={inputClass}>
          {locale === '' && <option value="">{t('storefront.companyLanguage')}</option>}
          {LOCALES.map(l => <option key={l} value={l}>{LOCALE_NAMES[l]}</option>)}
        </select>
        <p className="mt-1.5 text-2xs text-ink-4">{t('storefront.languageHelp')}</p>
      </div>
      <fieldset>
        <legend className={labelClass}>{t('storefront.otherLanguages')}</legend>
        <div className="flex flex-wrap gap-x-4 gap-y-1.5">
          {LOCALES.filter(l => l !== locale).map(l => (
            <label key={l} className="flex items-center gap-1.5 text-xs text-ink-2">
              <input type="checkbox" checked={others.includes(l)} onChange={e => toggle(l, e.target.checked)} />
              {LOCALE_NAMES[l]}
            </label>
          ))}
        </div>
        <p className="mt-1.5 text-2xs text-ink-4">{t('storefront.otherLanguagesHelp')}</p>
      </fieldset>
    </>
  )
}
