// Languages the admin speaks. Same tags as the storefront's.
export const LOCALES = ['en', 'pt-BR', 'nl'] as const
export type Locale = (typeof LOCALES)[number]
export const DEFAULT_LOCALE: Locale = 'en'

/** Each language's name in itself, for the language picker. */
export const LOCALE_NAMES: Record<Locale, string> = { en: 'English', 'pt-BR': 'Português (Brasil)', nl: 'Nederlands' }

/** Best supported match for a tag ("pt", "pt-PT", "nl-BE" …); English otherwise. */
export function resolveLocale(tag: string | null | undefined): Locale {
  if (!tag) return DEFAULT_LOCALE
  const exact = LOCALES.find(l => l.toLowerCase() === tag.toLowerCase())
  if (exact) return exact
  const language = tag.toLowerCase().split(/[-_]/)[0]
  return LOCALES.find(l => l.toLowerCase().split('-')[0] === language) ?? DEFAULT_LOCALE
}

/** Intl locale for numbers, prices and dates in each language. */
export const INTL_LOCALE: Record<Locale, string> = { en: 'en-GB', 'pt-BR': 'pt-BR', nl: 'nl-NL' }
