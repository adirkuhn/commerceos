import { describe, it, expect } from 'vitest'
import { unitsPerPurchasedItemOnLink } from '../procurementHelpers'

describe('unitsPerPurchasedItemOnLink', () => {
  it('books a kilo (weight pool) product per kg: 1000 grams per purchased item', () => {
    expect(unitsPerPurchasedItemOnLink({ unit_type: 'kilo' }, '1')).toBe('1000')
  })

  it('keeps a value someone already typed', () => {
    expect(unitsPerPurchasedItemOnLink({ unit_type: 'kilo' }, '5000')).toBe('5000')
    expect(unitsPerPurchasedItemOnLink({ unit_type: 'unit' }, '36')).toBe('36')
  })

  it('leaves every other product at its current value', () => {
    expect(unitsPerPurchasedItemOnLink({ unit_type: 'unit' }, '1')).toBe('1')
    expect(unitsPerPurchasedItemOnLink({}, '1')).toBe('1')
  })
})
