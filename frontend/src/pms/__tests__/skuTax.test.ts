import { describe, it, expect } from 'vitest'
import { computeTaxPreview, resolveSkuTaxRate } from '../skuTax'
import type { TaxClass, TaxRate } from '../../tax/types'

const classes: TaxClass[] = [
  { id: 'std', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
  { id: 'low', name: 'Reduced', is_default: false, created_at: '', updated_at: '' },
]
const rate = (tax_class_id: string, country: string, rate_basis_points: number): TaxRate =>
  ({ id: `${tax_class_id}-${country}`, tax_class_id, country, rate_basis_points, created_at: '', updated_at: '' })
const rates = [rate('std', 'NL', 2100), rate('low', 'NL', 900), rate('std', 'BE', 2100)]

describe('resolveSkuTaxRate', () => {
  it('uses the SKU\'s own tax class in the given country', () => {
    expect(resolveSkuTaxRate(rates, classes, 'low', 'NL')?.rate_basis_points).toBe(900)
  })

  it('falls back to the tenant default class when the SKU has none', () => {
    expect(resolveSkuTaxRate(rates, classes, null, 'NL')?.id).toBe('std-NL')
  })

  it('is undefined without a country, a default class, or a matching rate', () => {
    expect(resolveSkuTaxRate(rates, classes, null, '')).toBeUndefined()
    expect(resolveSkuTaxRate(rates, [classes[1]], null, 'NL')).toBeUndefined()
    expect(resolveSkuTaxRate(rates, classes, 'low', 'BE')).toBeUndefined()
  })
})

describe('computeTaxPreview', () => {
  it('treats the price as tax-inclusive when prices include tax', () => {
    expect(computeTaxPreview(1210, rate('std', 'NL', 2100), true)).toEqual({ exclusive: 10, inclusive: 12.1, ratePercent: '21.00', country: 'NL' })
  })

  it('treats the price as tax-exclusive otherwise', () => {
    expect(computeTaxPreview(1000, rate('std', 'NL', 2100), false)).toEqual({ exclusive: 10, inclusive: 12.1, ratePercent: '21.00', country: 'NL' })
  })

  it('is null without a rate or a price', () => {
    expect(computeTaxPreview(null, rate('std', 'NL', 2100), true)).toBeNull()
    expect(computeTaxPreview(1000, undefined, true)).toBeNull()
  })
})
