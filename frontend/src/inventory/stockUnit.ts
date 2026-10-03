import type { SKU } from '../pms/types'

// How a SKU's stock quantity is typed in vs stored. A weight pool (unit_type
// kilo) keeps its ledger in grams -- the backend's unit -- but people think
// and weigh in kilos, so every quantity field for it reads kg (decimals,
// comma or dot) and sends grams. Everything else is whole units.
export interface StockUnit {
  kind: 'unit' | 'kg'
  // The stored quantity for what was typed, or null when it isn't a number
  // of that unit.
  parse: (input: string) => number | null
  // A stored quantity as the user would type it.
  format: (stored: number) => string
  // For <input type="number" step=...>.
  step: string
}

const UNIT: StockUnit = {
  kind: 'unit',
  parse: input => (/^\s*-?\d+\s*$/.test(input) ? parseInt(input, 10) : null),
  format: stored => String(stored),
  step: '1',
}

const KG: StockUnit = {
  kind: 'kg',
  parse: input => {
    const normalized = input.trim().replace(',', '.')
    if (!/^-?\d+(\.\d+)?$/.test(normalized)) return null
    return Math.round(parseFloat(normalized) * 1000)
  },
  format: stored => String(stored / 1000),
  step: '0.001',
}

export function stockUnitFor(sku?: Pick<SKU, 'unit_type'>): StockUnit {
  return sku?.unit_type === 'kilo' ? KG : UNIT
}
