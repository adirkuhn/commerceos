import { describe, it, expect } from 'vitest'
import { addPackageLine, formatKg, isScaleSoldWeightPool, packagePriceMinorUnits, summarizePackages } from '../weightPackages'

describe('packagePriceMinorUnits', () => {
  it('rounds half up to the nearest minor unit, same as the backend', () => {
    // 29990/kg * 380g = 11396.2 -> 11396
    expect(packagePriceMinorUnits(29990, 380)).toBe(11396)
    // 29990/kg * 404g = 12115.96 -> 12116
    expect(packagePriceMinorUnits(29990, 404)).toBe(12116)
    // exactly .5 rounds up: 1000/kg * 0.5g... 1/kg * 500g = 0.5 -> 1
    expect(packagePriceMinorUnits(1, 500)).toBe(1)
  })
})

describe('addPackageLine', () => {
  it('appends a new weight at the end', () => {
    expect(addPackageLine([{ weightGrams: 430, quantity: 1 }], { weightGrams: 380, quantity: 10 }))
      .toEqual([{ weightGrams: 430, quantity: 1 }, { weightGrams: 380, quantity: 10 }])
  })

  it('adds to the quantity of a weight already in the list instead of duplicating it', () => {
    expect(addPackageLine([{ weightGrams: 380, quantity: 3 }, { weightGrams: 430, quantity: 1 }], { weightGrams: 380, quantity: 2 }))
      .toEqual([{ weightGrams: 380, quantity: 5 }, { weightGrams: 430, quantity: 1 }])
  })

  it('ignores a non-positive or fractional weight or quantity', () => {
    const lines = [{ weightGrams: 380, quantity: 1 }]
    expect(addPackageLine(lines, { weightGrams: 0, quantity: 1 })).toBe(lines)
    expect(addPackageLine(lines, { weightGrams: 380, quantity: 0 })).toBe(lines)
    expect(addPackageLine(lines, { weightGrams: 380.5, quantity: 1 })).toBe(lines)
    expect(addPackageLine(lines, { weightGrams: 380, quantity: -2 })).toBe(lines)
  })
})

describe('summarizePackages', () => {
  it('counts distinct weights, packages and total grams', () => {
    expect(summarizePackages([
      { weightGrams: 430, quantity: 1 }, { weightGrams: 540, quantity: 4 },
      { weightGrams: 412, quantity: 1 }, { weightGrams: 380, quantity: 10 },
    ])).toEqual({ weights: 4, packages: 16, totalGrams: 430 + 2160 + 412 + 3800 })
  })

  it('is all zeros for an empty list', () => {
    expect(summarizePackages([])).toEqual({ weights: 0, packages: 0, totalGrams: 0 })
  })
})

describe('isScaleSoldWeightPool', () => {
  const pool = { unit_type: 'kilo' as const, pricing_mode: 'per_weight' as const, scale_code: '2994721', price_per_kilo_minor_units: 29990 }

  it('is true for a kilo SKU priced per weight with a scale code and kilo price', () => {
    expect(isScaleSoldWeightPool(pool)).toBe(true)
  })

  it('is false when any of those is missing', () => {
    expect(isScaleSoldWeightPool({ ...pool, unit_type: 'unit' })).toBe(false)
    expect(isScaleSoldWeightPool({ ...pool, pricing_mode: 'fixed' })).toBe(false)
    expect(isScaleSoldWeightPool({ ...pool, scale_code: null })).toBe(false)
    expect(isScaleSoldWeightPool({ ...pool, scale_code: '' })).toBe(false)
    expect(isScaleSoldWeightPool({ ...pool, price_per_kilo_minor_units: null })).toBe(false)
  })
})

describe('formatKg', () => {
  it('renders grams as kilos with two decimals', () => {
    expect(formatKg(4990)).toBe('4.99 kg')
    expect(formatKg(50000)).toBe('50.00 kg')
    expect(formatKg(0)).toBe('0.00 kg')
  })
})
