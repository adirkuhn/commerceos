import { describe, it, expect } from 'vitest'
import { EMPTY_FLAT_SKU_FORM, flatSkuBasePayload, flatSkuCreatePayload, flatSkuFormFromSKU, flatSkuUpdatePayload, type FlatSkuForm } from '../flatSkuForm'
import type { SKU } from '../types'

const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'A-1', barcode: null, price_minor_units: 1999,
  currency_code: 'EUR', is_virtual: false, status: 'active', created_at: '', updated_at: '',
}

const filled: FlatSkuForm = {
  ...EMPTY_FLAT_SKU_FORM,
  skuCode: 'A-1', barcode: '', price: '19.99', salePrice: '', saleEndDate: '', currency: 'EUR',
  weightGrams: '250', lengthMM: '', widthMM: 'x', heightMM: '3',
  unitType: 'box', portionedFromSKUID: 'pool-1', caseOfSKUID: 'unit-1', unitsPerCase: '12',
  scaleCode: '', pricingMode: 'fixed', pricePerKilo: '9.99', taxClassId: 'tc-1',
}

describe('flatSkuFormFromSKU', () => {
  it('maps a minimal SKU to blank/default inputs', () => {
    expect(flatSkuFormFromSKU(sku)).toEqual({
      skuCode: 'A-1', barcode: '', price: '19.99', salePrice: '', saleEndDate: '', currency: 'EUR', skuStatus: 'active',
      isVirtual: false, weightGrams: '', lengthMM: '', widthMM: '', heightMM: '',
      caseOfSKUID: null, caseOfSKULabel: '', unitsPerCase: '', unitType: 'unit', portionedFromSKUID: null, portionedFromLabel: '',
      scaleCode: '', pricingMode: 'fixed', pricePerKilo: '', taxClassId: null,
    })
  })

  it('names the linked case / weight-pool SKU from the product detail, not its id', () => {
    const unit = { id: 'u', product_name: 'Refresco em pó Uva', sku_code: 'REF-UVA', barcode: '7891132082346' }
    const f = flatSkuFormFromSKU({ ...sku, case_of_sku_id: 'u', case_of_sku: unit, portioned_from_sku_id: 'u', portioned_from_sku: unit })
    expect(f.caseOfSKULabel).toBe('Refresco em pó Uva · EAN 7891132082346')
    expect(f.portionedFromLabel).toBe('Refresco em pó Uva · EAN 7891132082346')
  })

  it('formats every set optional field as its input string', () => {
    const f = flatSkuFormFromSKU({
      ...sku, barcode: '123', sale_price_minor_units: 1500, sale_end_date: '2026-12-31', weight_grams: 0, length_mm: 10,
      case_of_sku_id: 'u', units_per_case: 6, unit_type: 'kilo', scale_code: '2900633', pricing_mode: 'per_weight',
      price_per_kilo_minor_units: 2999, tax_class_id: 'tc',
    })
    expect(f).toMatchObject({
      barcode: '123', salePrice: '15.00', saleEndDate: '2026-12-31', weightGrams: '0', lengthMM: '10',
      caseOfSKUID: 'u', unitsPerCase: '6', unitType: 'kilo', scaleCode: '2900633', pricingMode: 'per_weight', pricePerKilo: '29.99', taxClassId: 'tc',
    })
  })
})

describe('flat SKU payloads', () => {
  it('base payload parses inputs and blanks to undefined/null', () => {
    expect(flatSkuBasePayload(filled)).toEqual({
      sku_code: 'A-1', barcode: undefined, price_minor_units: 1999, sale_price_minor_units: null, sale_end_date: null,
      currency_code: 'EUR', is_virtual: false, weight_grams: 250, length_mm: undefined, width_mm: undefined, height_mm: 3,
      scale_code: null, pricing_mode: 'fixed', price_per_kilo_minor_units: null, tax_class_id: 'tc-1', status: 'active',
    })
  })

  it('a virtual SKU sends no dimensions; per-weight pricing sends its kilo price', () => {
    const p = flatSkuBasePayload({ ...filled, isVirtual: true, pricingMode: 'per_weight' })
    expect(p).toMatchObject({ is_virtual: true, weight_grams: undefined, height_mm: undefined, price_per_kilo_minor_units: 999 })
  })

  it('create payload adds the weight-pool link but not the case link', () => {
    const p = flatSkuCreatePayload(filled)
    expect(p).toMatchObject({ unit_type: 'box', portioned_from_sku_id: 'pool-1' })
    expect(p).not.toHaveProperty('case_of_sku_id')
  })

  it('update payload adds case link, weight-pool link and passes tracks_expiry through', () => {
    expect(flatSkuUpdatePayload(filled, { ...sku, tracks_expiry: true })).toMatchObject({
      tracks_expiry: true, case_of_sku_id: 'unit-1', units_per_case: 12, unit_type: 'box', portioned_from_sku_id: 'pool-1',
    })
  })

  it('update payload drops units_per_case when there is no case link', () => {
    expect(flatSkuUpdatePayload({ ...filled, caseOfSKUID: null }, sku)).toMatchObject({ case_of_sku_id: null, units_per_case: null })
    expect(flatSkuUpdatePayload({ ...filled, unitsPerCase: '' }, sku)).toMatchObject({ units_per_case: null })
  })
})
