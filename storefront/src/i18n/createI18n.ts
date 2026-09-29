import { createInstance, type i18n, type Resource } from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DEFAULT_LOCALE, type Locale } from './locale'
import { resources as shopResources } from './resources'

/** A fresh i18next instance for one language. One per SSR request and one
 *  in the browser, initialised synchronously from bundled texts (the shop
 *  namespace unless given more, e.g. the editor's). */
export function createI18n(locale: Locale, resources: Partial<Record<Locale, Resource[string]>> = shopResources): i18n {
  const instance = createInstance()
  void instance.use(initReactI18next).init({
    lng: locale,
    fallbackLng: DEFAULT_LOCALE,
    resources: resources as Resource,
    ns: Object.keys(resources[DEFAULT_LOCALE] ?? {}),
    defaultNS: 'shop',
    initAsync: false,
    // A blank translation falls back to English rather than showing nothing.
    returnEmptyString: false,
    // React escapes output already.
    interpolation: { escapeValue: false },
  })
  return instance
}
