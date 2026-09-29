import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DEFAULT_LOCALE, resolveLocale, type Locale } from './locale'
import { NAMESPACES, resources } from './resources'

// The admin's one i18next instance. The language is the signed-in user's
// choice, kept in this browser; first visit follows the browser language.

const STORAGE_KEY = 'cos-admin-language'

function storedLocale(): Locale | undefined {
  try {
    const v = window.localStorage.getItem(STORAGE_KEY)
    return v ? resolveLocale(v) : undefined
  } catch {
    return undefined
  }
}

export function initialLocale(): Locale {
  if (typeof window === 'undefined') return DEFAULT_LOCALE
  return storedLocale() ?? resolveLocale(window.navigator.language)
}

export function initI18n(locale: Locale = initialLocale()) {
  void i18next.use(initReactI18next).init({
    lng: locale,
    fallbackLng: DEFAULT_LOCALE,
    resources,
    ns: NAMESPACES,
    defaultNS: 'common',
    initAsync: false,
    // A blank translation falls back to English rather than showing nothing.
    returnEmptyString: false,
    interpolation: { escapeValue: false },
  })
  if (typeof document !== 'undefined') document.documentElement.lang = locale
  return i18next
}

/** Switches the admin language now and remembers it in this browser. */
export async function setLanguage(locale: Locale) {
  await i18next.changeLanguage(locale)
  document.documentElement.lang = locale
  try {
    window.localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // not remembered; still switched for this visit
  }
}

export const currentLocale = (): Locale => resolveLocale(i18next.language)
