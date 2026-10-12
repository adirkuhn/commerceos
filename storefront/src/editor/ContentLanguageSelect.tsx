import { useTranslation } from 'react-i18next'
import { LOCALE_NAMES, type Locale } from '../i18n/locale'
import type { StoreLanguages } from '../runtime/storeLocale'

/** Which language the merchant's texts are being edited in: "" is the
 *  store's default (the component settings themselves), any other offered
 *  language edits that language's translations. */
export function ContentLanguageSelect({ languages, value, onChange }: {
  languages: StoreLanguages | undefined
  value: Locale | ''
  onChange: (l: Locale | '') => void
}) {
  const { t } = useTranslation('editor')
  if (!languages || languages.offered.length < 2) return null
  return (
    <>
      <label className="text-sm text-ed-ink-3" htmlFor="content-language">{t('toolbar.textsIn')}</label>
      <select id="content-language" className="rounded-ed border border-ed-line bg-ed-sunken px-2.5 py-1.5 text-sm text-ed-ink focus:outline-none focus:ring-2 focus:ring-ed-accent/60" value={value}
        onChange={e => onChange(e.target.value as Locale | '')}>
        <option value="">{t('toolbar.defaultLanguage', { language: LOCALE_NAMES[languages.defaultLocale] })}</option>
        {languages.offered.filter(l => l !== languages.defaultLocale).map(l => <option key={l} value={l}>{LOCALE_NAMES[l]}</option>)}
      </select>
    </>
  )
}
