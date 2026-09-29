import { LOCALES, resolveLocale, type Locale } from '../i18n/locale'

// Which language a store page is in. A store has a default language (plain
// URLs: /store/{key}/products/x) and may offer others, each under its own
// prefix (/store/{key}/nl/products/x) so every language has its own
// crawlable, shareable URL. A shopper's pick is also remembered in a
// cookie, so a plain URL later takes them back to their language.

export interface StoreLanguages {
  defaultLocale: Locale
  /** Every language the store offers, the default first. */
  offered: Locale[]
}

const isLocale = (tag: string): tag is Locale => (LOCALES as readonly string[]).includes(tag)

export function storeLanguages(store: { locale?: string; locales?: string[] } | null | undefined): StoreLanguages {
  const defaultLocale = resolveLocale(store?.locale)
  const offered: Locale[] = [defaultLocale]
  for (const tag of store?.locales ?? []) {
    if (isLocale(tag) && !offered.includes(tag)) offered.push(tag)
  }
  return { defaultLocale, offered }
}

/** Splits a path below /store/{key} into its language and the page path. */
export function splitLocale(path: string, langs: StoreLanguages): { locale: Locale; path: string; prefixed: boolean } {
  const m = /^\/([^/]+)(\/.*)?$/.exec(path)
  const first = m?.[1] ?? ''
  if (isLocale(first) && first !== langs.defaultLocale && langs.offered.includes(first)) {
    return { locale: first, path: m?.[2] ?? '/', prefixed: true }
  }
  return { locale: langs.defaultLocale, path, prefixed: false }
}

/** Builds store links ("products/x") in one language. */
export function storePathFor(storeKey: string, locale: Locale, langs: StoreLanguages): (p: string) => string {
  const prefix = locale === langs.defaultLocale ? '' : `${locale}/`
  return p => `/store/${encodeURIComponent(storeKey)}/${prefix}${p}`
}

export const SHOP_LANGUAGE_COOKIE = 'cos_shop_lang'
const ONE_YEAR = 60 * 60 * 24 * 365

/** Set-Cookie / document.cookie value remembering a shopper's language
 *  for this store. */
export function languageCookie(storeKey: string, locale: Locale): string {
  return `${SHOP_LANGUAGE_COOKIE}=${locale}; Path=/store/${encodeURIComponent(storeKey)}/; Max-Age=${ONE_YEAR}; SameSite=Lax`
}

export function localeFromCookie(cookieHeader: string | undefined, langs: StoreLanguages): Locale | undefined {
  const m = new RegExp(`(?:^|;\\s*)${SHOP_LANGUAGE_COOKIE}=([^;]*)`).exec(cookieHeader ?? '')
  const tag = m ? decodeURIComponent(m[1]) : ''
  return isLocale(tag) && langs.offered.includes(tag) ? tag : undefined
}
