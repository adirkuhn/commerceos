import { describe, it, expect } from 'vitest'
import { emptyInput, fromMinorUnits, slugify, toMinorUnits, validationMessage, withType } from '../form'

describe('collection form helpers', () => {
  it('slugifies names for the storefront URL', () => {
    expect(slugify('Summer Sale 2026!')).toBe('summer-sale-2026')
    expect(slugify('  Crème brûlée  ')).toBe('creme-brulee')
    expect(slugify('x'.repeat(80))).toHaveLength(64)
  })

  it('switching to smart adds a first condition and a non-manual sort; back to manual drops rules', () => {
    const smart = withType({ ...emptyInput(), name: 'A', slug: 'a' }, 'smart')
    expect(smart.rules).toHaveLength(1)
    expect(smart.sort).toBe('newest')
    expect(withType(smart, 'manual').rules).toEqual([])
  })

  it('explains what to fix', () => {
    expect(validationMessage(emptyInput())).toMatch(/name/)
    expect(validationMessage({ ...emptyInput(), name: 'A', slug: 'Bad Slug' })).toMatch(/URL/)
    const smart = withType({ ...emptyInput(), name: 'A', slug: 'a' }, 'smart')
    expect(validationMessage(smart)).toMatch(/category/)
    expect(validationMessage({ ...smart, rules: [{ field: 'price', min_minor_units: 500, max_minor_units: 100 }] })).toMatch(/minimum/)
    expect(validationMessage({ ...smart, rules: [{ field: 'on_sale' }] })).toBe('')
  })

  it('converts money between what merchants type and minor units', () => {
    expect(toMinorUnits('12,50')).toBe(1250)
    expect(toMinorUnits('')).toBeNull()
    expect(toMinorUnits('-1')).toBeNull()
    expect(fromMinorUnits(1250)).toBe('12.50')
    expect(fromMinorUnits(null)).toBe('')
  })
})
