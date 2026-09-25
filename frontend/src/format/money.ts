// formatMoney renders an integer minor-units amount (cents) as a localized
// currency string -- the one formatter every screen uses, so an amount
// looks the same everywhere. fallbackCurrency covers a record with no
// currency code yet; a code Intl doesn't know falls back to "12.50 XYZ"
// rather than throwing.
export function formatMoney(minorUnits: number, currencyCode: string, fallbackCurrency = 'USD'): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || fallbackCurrency }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}
