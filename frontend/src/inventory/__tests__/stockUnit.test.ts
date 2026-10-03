import { describe, it, expect } from 'vitest'
import { stockUnitFor } from '../stockUnit'

describe('stockUnitFor', () => {
  it('reads a kilo SKU\'s quantity in kg and stores grams', () => {
    const kg = stockUnitFor({ unit_type: 'kilo' })
    expect(kg.kind).toBe('kg')
    expect(kg.parse('50')).toBe(50000)
    expect(kg.parse('0.5')).toBe(500)
    expect(kg.parse('0,5')).toBe(500)
    expect(kg.parse('12.345')).toBe(12345)
    expect(kg.parse('-1.2')).toBe(-1200)
    expect(kg.parse('')).toBeNull()
    expect(kg.parse('abc')).toBeNull()
    expect(kg.format(50000)).toBe('50')
    expect(kg.format(1250)).toBe('1.25')
  })

  it('keeps whole units for every other SKU', () => {
    for (const unit of [stockUnitFor({ unit_type: 'unit' }), stockUnitFor({ unit_type: 'box' }), stockUnitFor({}), stockUnitFor(undefined)]) {
      expect(unit.kind).toBe('unit')
      expect(unit.parse('50')).toBe(50)
      expect(unit.parse('-3')).toBe(-3)
      expect(unit.parse('1.5')).toBeNull()
      expect(unit.parse('')).toBeNull()
      expect(unit.format(50)).toBe('50')
    }
  })
})
