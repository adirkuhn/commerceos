import { describe, it, expect } from 'vitest'
import { formatPrice } from '../money'

describe('formatPrice', () => {
  it('formats minor units in the currency', () => {
    expect(formatPrice(2999, 'EUR')).toBe('€ 29,99')
    expect(formatPrice(100000, 'EUR')).toBe('€ 1.000,00')
  })

  it('respects currencies without minor units', () => {
    expect(formatPrice(500, 'JPY')).toBe('JP¥ 500')
  })

  it('returns empty for missing prices', () => {
    expect(formatPrice(null, 'EUR')).toBe('')
    expect(formatPrice(100, null)).toBe('')
  })
})
