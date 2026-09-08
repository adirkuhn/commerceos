export function formatMoney(minorUnits: number, currencyCode: string): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode }).format(minorUnits / 100)
}
