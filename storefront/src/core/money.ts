// Store-wide display locale until stores get a locale setting (CommerceOS
// is Netherlands-first). Server and browser must format identically, or
// hydration mismatches.
export const STORE_LOCALE = 'nl-NL'

/** Formats integer minor units (e.g. cents) as a price. */
export function formatPrice(minorUnits: number | null | undefined, currency: string | null | undefined): string {
  if (minorUnits == null || !currency) return ''
  const fmt = new Intl.NumberFormat(STORE_LOCALE, { style: 'currency', currency })
  const digits = fmt.resolvedOptions().maximumFractionDigits ?? 2
  return fmt.format(minorUnits / 10 ** digits).replace(/ /g, ' ')
}
