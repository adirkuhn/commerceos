import type { SKU } from './types'

// One row of "N packages of W grams" weighed off a weight-pool SKU.
export interface PackageLine {
  weightGrams: number
  quantity: number
}

// Same "round-half-up to the nearest minor unit" the backend applies
// (lineTotalForScannedWeight) -- display only, the package SKU's real
// price always comes from the server.
export function packagePriceMinorUnits(pricePerKiloMinorUnits: number, weightGrams: number): number {
  return Math.floor((pricePerKiloMinorUnits * weightGrams + 500) / 1000)
}

function isPositiveWhole(n: number): boolean {
  return Number.isInteger(n) && n > 0
}

// Adds a line, folding it into an existing line of the same weight -- the
// backend merges them the same way (one package SKU per weight). An
// invalid line leaves the list untouched.
export function addPackageLine(lines: PackageLine[], line: PackageLine): PackageLine[] {
  if (!isPositiveWhole(line.weightGrams) || !isPositiveWhole(line.quantity)) return lines
  const i = lines.findIndex(l => l.weightGrams === line.weightGrams)
  if (i === -1) return [...lines, line]
  return lines.map((l, j) => (j === i ? { ...l, quantity: l.quantity + line.quantity } : l))
}

export function summarizePackages(lines: PackageLine[]): { weights: number; packages: number; totalGrams: number } {
  return lines.reduce(
    (acc, l) => ({ weights: acc.weights + 1, packages: acc.packages + l.quantity, totalGrams: acc.totalGrams + l.weightGrams * l.quantity }),
    { weights: 0, packages: 0, totalGrams: 0 },
  )
}

// Can this SKU be split into priced, scale-barcoded packages? Mirrors the
// backend's own guard (ErrBulkSKUNotWeightPool).
export function isScaleSoldWeightPool(sku: Pick<SKU, 'unit_type' | 'pricing_mode' | 'scale_code' | 'price_per_kilo_minor_units'>): boolean {
  return sku.unit_type === 'kilo' && sku.pricing_mode === 'per_weight' && !!sku.scale_code && sku.price_per_kilo_minor_units != null
}

// A weight pool's stock is held in grams -- shown as kilos.
export function formatKg(grams: number): string {
  return `${(grams / 1000).toFixed(2)} kg`
}
