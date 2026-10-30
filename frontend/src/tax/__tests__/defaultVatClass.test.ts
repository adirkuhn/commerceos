import { describe, it, expect } from 'vitest'
import { defaultVatProblem } from '../defaultVatClass'
import type { TaxClass, TaxRate } from '../types'

const cls = (id: string, isDefault = false) => ({ id, name: id, is_default: isDefault }) as TaxClass
const rate = (classId: string, country: string, bp: number) => ({ id: `${classId}-${country}`, tax_class_id: classId, country, rate_basis_points: bp }) as TaxRate

describe('defaultVatProblem', () => {
  it('none when the default class has a rate for the tax country', () => {
    expect(defaultVatProblem([cls('reduced', true)], [rate('reduced', 'NL', 900)], 'NL')).toBeNull()
  })

  it('a configured 0% rate is a real rate, not a problem', () => {
    expect(defaultVatProblem([cls('zero', true)], [rate('zero', 'NL', 0)], 'nl')).toBeNull()
  })

  it('no default class', () => {
    expect(defaultVatProblem([cls('standard'), cls('reduced')], [rate('reduced', 'NL', 900)], 'NL')).toBe('noDefault')
  })

  it('the default class has no rate for the tax country', () => {
    expect(defaultVatProblem([cls('standard', true)], [rate('standard', 'BE', 2100)], 'NL')).toBe('noRateForCountry')
  })

  it('no tax classes at all', () => {
    expect(defaultVatProblem([], [], 'NL')).toBe('noDefault')
  })
})
