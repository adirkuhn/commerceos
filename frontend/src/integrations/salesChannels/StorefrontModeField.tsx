import { useTranslation } from 'react-i18next'
import type { StorefrontLocale, StorefrontMode } from '../types'
import { LOCALES, LOCALE_NAMES } from '../../i18n/locale'

interface Props {
  value: StorefrontMode
  onChange: (mode: StorefrontMode) => void
  signInUrl: string
  onSignInUrlChange: (url: string) => void
  locale: StorefrontLocale
  onLocaleChange: (locale: StorefrontLocale) => void
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const labelClass = 'block text-2xs text-ink-4 uppercase tracking-wider mb-1.5'

export default function StorefrontModeField({ value, onChange, signInUrl, onSignInUrlChange, locale, onLocaleChange }: Props) {
  const { t } = useTranslation('channels')
  const options: { value: StorefrontMode; label: string; help: string }[] = [
    { value: 'headless', label: t('storefront.headless'), help: t('storefront.headlessHelp') },
    { value: 'theme', label: t('storefront.theme'), help: t('storefront.themeHelp') },
  ]
  const help = options.find(o => o.value === value)?.help
  return (
    <>
      <div>
        <label htmlFor="channel-storefront-mode" className={labelClass}>{t('storefront.label')}</label>
        <select id="channel-storefront-mode" value={value} onChange={e => onChange(e.target.value as StorefrontMode)} className={inputClass}>
          {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        {help && <p className="mt-1.5 text-2xs text-ink-4">{help}</p>}
      </div>
      <div>
        <label htmlFor="channel-storefront-locale" className={labelClass}>{t('storefront.language')}</label>
        <select id="channel-storefront-locale" value={locale} onChange={e => onLocaleChange(e.target.value as StorefrontLocale)} className={inputClass}>
          {LOCALES.map(l => <option key={l} value={l}>{LOCALE_NAMES[l]}</option>)}
        </select>
        <p className="mt-1.5 text-2xs text-ink-4">{t('storefront.languageHelp')}</p>
      </div>
      {value === 'headless' && (
        <div>
          <label htmlFor="channel-sign-in-url" className={labelClass}>{t('storefront.signInPage')}</label>
          <input id="channel-sign-in-url" type="url" inputMode="url" value={signInUrl} maxLength={2000}
            placeholder={'https://shop.example.com/account/verify'} onChange={e => onSignInUrlChange(e.target.value)} className={inputClass} />
          <p className="mt-1.5 text-2xs text-ink-4">{t('storefront.signInPageHelp')}</p>
        </div>
      )}
    </>
  )
}
