import { describe, expect, it } from 'vitest'
import { formatProductStock } from '../productStock'

const unit = (id: string) => ({ id, unit_type: 'unit' as const })
const kilo = (id: string) => ({ id, unit_type: 'kilo' as const })

describe('formatProductStock', () => {
  it('sums whole units as a plain count', () => {
    expect(formatProductStock([unit('a'), unit('b')], { a: 3, b: 4 })).toBe('7')
  })

  it('shows a weight pool (stored in grams) in kg', () => {
    expect(formatProductStock([kilo('k')], { k: 6000 })).toBe('6.00kg')
  })

  it('never adds grams to pieces: a pool with unit packages shows both', () => {
    expect(formatProductStock([kilo('k'), unit('p1'), unit('p2')], { k: 1500, p1: 2, p2: 3 })).toBe('1.50kg · 5')
  })

  it('counts a SKU with no level loaded as zero', () => {
    expect(formatProductStock([unit('a')], {})).toBe('0')
    expect(formatProductStock([], {})).toBe('0')
  })
})
