import { useTranslation } from 'react-i18next'
import { LOCALE_NAMES, type Locale } from '../../i18n/locale'
import { useStoreLanguage } from '../../runtime/page/StoreLanguageContext'
import { languageCookie } from '../../runtime/storeLocale'
import { EditorHint } from '../EditorHint'
import type { RendererProps } from '../types'
import type { LanguageSwitcherProps } from './languageSwitcher'

const code = (l: Locale) => l.split('-')[0].toUpperCase()

function remember(storeKey: string, locale: Locale) {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = languageCookie(storeKey, locale) + secure
}

/** Plain links (crawlable, work without JavaScript); a click also
 *  remembers the pick so the store's plain URLs open in it next time. */
export function LanguageSwitcherView({ node, props, className }: RendererProps<LanguageSwitcherProps>) {
  const languages = useStoreLanguage()
  const { t } = useTranslation()
  if (!languages) return <EditorHint nodeId={node.id} className={className} hint="languageSwitcher" />
  if (languages.offered.length < 2) return null
  return (
    <nav className={`cos-language-switcher ${className}`} data-node={node.id} aria-label={t('nav.language')}>
      <ul>
        {languages.offered.map(l => (
          <li key={l}>
            <a href={languages.hrefFor(l)} hrefLang={l} lang={l} aria-label={LOCALE_NAMES[l]}
              aria-current={l === languages.locale ? 'true' : undefined} onClick={() => remember(languages.storeKey, l)}>
              {props.display === 'codes' ? code(l) : LOCALE_NAMES[l]}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}
