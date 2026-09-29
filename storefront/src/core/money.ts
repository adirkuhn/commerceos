// Server and browser must format identically (same Intl locale), or
// hydration mismatches. Components format through the store's i18n
// (useShopI18n().price); the default is the Netherlands-first fallback.
const DEFAULT_INTL_LOCALE = 'nl-NL'

/** Formats integer minor units (e.g. cents) as a price. */
export function formatPrice(minorUnits: number | null | undefined, currency: string | null | undefined, intlLocale = DEFAULT_INTL_LOCALE): string {
  if (minorUnits == null || !currency) return ''
  const fmt = new Intl.NumberFormat(intlLocale, { style: 'currency', currency })
  const digits = fmt.resolvedOptions().maximumFractionDigits ?? 2
  return fmt.format(minorUnits / 10 ** digits).replace(/ /g, ' ')
}

/** Minor units as a plain decimal amount ("40.00"), e.g. for schema.org. */
export function decimalAmount(minorUnits: number, currency: string): string {
  const digits = new Intl.NumberFormat('en', { style: 'currency', currency }).resolvedOptions().maximumFractionDigits ?? 2
  return (minorUnits / 10 ** digits).toFixed(digits)
}
