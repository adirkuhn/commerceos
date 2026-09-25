export function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
}

export function formatMinorUnitsAsInput(minorUnits: number): string {
  return (minorUnits / 100).toFixed(2)
}

export function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}
