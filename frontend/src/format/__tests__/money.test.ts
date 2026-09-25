import { describe, it, expect } from 'vitest'
import { formatMoney } from '../money'

describe('formatMoney', () => {
  it('formats minor units as a localized currency amount', () => {
    expect(formatMoney(13000, 'EUR')).toBe(new Intl.NumberFormat(undefined, { style: 'currency', currency: 'EUR' }).format(130))
  })

  it('uses the fallback currency when none is given (USD by default)', () => {
    expect(formatMoney(100, '')).toBe(new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD' }).format(1))
    expect(formatMoney(100, '', 'EUR')).toBe(new Intl.NumberFormat(undefined, { style: 'currency', currency: 'EUR' }).format(1))
  })

  it('falls back to plain "<amount> <code>" for a code Intl rejects', () => {
    expect(formatMoney(1250, 'NOT-A-CODE')).toBe('12.50 NOT-A-CODE')
  })
})
