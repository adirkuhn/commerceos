import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DEFAULT_LOCALE, resolveLocale, type Locale } from './locale'
import { NAMESPACES, resources } from './resources'

// The admin's one i18next instance. The language is the user's choice, kept
// in a cookie; without one it is the company language (Settings →
// Localization), and the browser language until that has loaded.

export const LANGUAGE_COOKIE = 'cos_lang'
const ONE_YEAR = 60 * 60 * 24 * 365

function storedLocale(): Locale | undefined {
  const match = document.cookie.match(new RegExp(`(?:^|; )${LANGUAGE_COOKIE}=([^;]*)`))
  return match ? resolveLocale(decodeURIComponent(match[1])) : undefined
}

function rememberLocale(locale: Locale) {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `${LANGUAGE_COOKIE}=${encodeURIComponent(locale)}; Path=/; Max-Age=${ONE_YEAR}; SameSite=Lax${secure}`
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

/** Switches the admin language now and remembers it (cookie). */
export async function setLanguage(locale: Locale) {
  await i18next.changeLanguage(locale)
  document.documentElement.lang = locale
  rememberLocale(locale)
}

/**
 * Follows the company language unless the user picked their own. Not
 * remembered: a later company change applies again.
 */
export async function applyCompanyLanguage(tag: string) {
  if (!tag || storedLocale()) return
  const locale = resolveLocale(tag)
  await i18next.changeLanguage(locale)
  document.documentElement.lang = locale
}

export const currentLocale = (): Locale => resolveLocale(i18next.language)
