import { describe, it, expect } from 'vitest'
import { dimensionInputsFromSKU, dimensionsPayload, skuPassThroughPayload } from '../skuPayloads'
import type { SKU } from '../types'

const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'A-1', barcode: null, price_minor_units: 1999,
  sale_price_minor_units: 1500, sale_end_date: '2026-12-31', currency_code: 'EUR', weight_grams: 100, length_mm: null,
  is_virtual: false, tracks_expiry: true, case_of_sku_id: 'u', units_per_case: 6, unit_type: 'box', portioned_from_sku_id: null,
  scale_code: '2900633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 2999, tax_class_id: 'tc-1',
  status: 'inactive', created_at: '', updated_at: '',
}

describe('skuPassThroughPayload', () => {
  it('sends every field back as loaded, except tax_class_id', () => {
    const p = skuPassThroughPayload(sku)
    expect(p).toEqual({
      sku_code: 'A-1', barcode: undefined, price_minor_units: 1999, sale_price_minor_units: 1500, sale_end_date: '2026-12-31',
      currency_code: 'EUR', weight_grams: 100, length_mm: null, width_mm: undefined, height_mm: undefined, is_virtual: false,
      tracks_expiry: true, case_of_sku_id: 'u', units_per_case: 6, unit_type: 'box', portioned_from_sku_id: null,
      scale_code: '2900633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 2999, status: 'inactive',
    })
    expect(p).not.toHaveProperty('tax_class_id')
  })
})

describe('dimensions', () => {
  it('parses dimension inputs, dropping blanks and junk', () => {
    expect(dimensionsPayload(false, { weightGrams: '10', lengthMM: '', widthMM: 'abc', heightMM: '0' }))
      .toEqual({ weight_grams: 10, length_mm: undefined, width_mm: undefined, height_mm: 0 })
  })

  it('a virtual SKU never carries dimensions', () => {
    expect(dimensionsPayload(true, { weightGrams: '10', lengthMM: '1', widthMM: '2', heightMM: '3' }))
      .toEqual({ weight_grams: undefined, length_mm: undefined, width_mm: undefined, height_mm: undefined })
  })

  it('turns a SKU\'s dimensions back into inputs', () => {
    expect(dimensionInputsFromSKU(sku)).toEqual({ weightGrams: '100', lengthMM: '', widthMM: '', heightMM: '' })
  })
})
