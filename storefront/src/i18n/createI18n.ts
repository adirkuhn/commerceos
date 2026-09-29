import { createInstance, type i18n } from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DEFAULT_LOCALE, type Locale } from './locale'
import { NAMESPACES, resources } from './resources'

/** A fresh i18next instance for one store language. One per SSR request
 *  and one in the browser, initialised synchronously from bundled texts. */
export function createI18n(locale: Locale): i18n {
  const instance = createInstance()
  void instance.use(initReactI18next).init({
    lng: locale,
    fallbackLng: DEFAULT_LOCALE,
    resources,
    ns: [...NAMESPACES],
    defaultNS: 'shop',
    initAsync: false,
    // A blank translation falls back to English rather than showing nothing.
    returnEmptyString: false,
    // React escapes output already.
    interpolation: { escapeValue: false },
  })
  return instance
}
