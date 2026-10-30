import { describe, it, expect } from 'vitest'
import { taxSetupGap } from '../taxSetupGap'
import type { TaxClass, TaxRate } from '../types'

const cls = (id: string, is_default: boolean): TaxClass => ({ id, name: id, is_default, created_at: '', updated_at: '' } as TaxClass)
const rate = (tax_class_id: string, country: string): TaxRate => ({ id: `${tax_class_id}-${country}`, tax_class_id, country, rate_basis_points: 900 } as TaxRate)

describe('taxSetupGap', () => {
  it('is null when the default class has a rate in the tax country', () => {
    expect(taxSetupGap([cls('std', true)], [rate('std', 'NL')], 'NL')).toBeNull()
  })

  it('names a missing tax country first', () => {
    expect(taxSetupGap([cls('std', true)], [rate('std', 'NL')], '')).toBe('country')
  })

  it('names a missing default class', () => {
    expect(taxSetupGap([cls('std', false)], [rate('std', 'NL')], 'NL')).toBe('defaultClass')
  })

  it('names a missing rate for the country', () => {
    expect(taxSetupGap([cls('std', true)], [rate('std', 'BE')], 'NL')).toBe('rate')
  })
})
