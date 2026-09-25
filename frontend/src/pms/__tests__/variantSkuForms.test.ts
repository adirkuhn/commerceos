import { describe, it, expect } from 'vitest'
import { emptyVariantSkuDraft, variantEditFormFrom, variantEditSkuPayload, variantSkuAddPayload } from '../variantSkuForms'
import type { SKU, VariantWithSKUs } from '../types'

const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'A-1', barcode: '123', price_minor_units: 1999,
  sale_price_minor_units: 1500, currency_code: 'EUR', weight_grams: 300, is_virtual: false, tracks_expiry: true,
  unit_type: 'kilo', scale_code: '2900633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 2999, tax_class_id: 'tc',
  status: 'inactive', created_at: '', updated_at: '',
}
const variant: VariantWithSKUs = { id: 'var-1', tenant_id: 't1', product_id: 'p', name: 'Large', position: 0, created_at: '', updated_at: '', skus: [sku] }

describe('variantSkuAddPayload', () => {
  it('builds the add payload without a tax class', () => {
    const p = variantSkuAddPayload({ ...emptyVariantSkuDraft('EUR'), skuCode: 'L-1', price: '2', weightGrams: '5', unitType: 'box' })
    expect(p).toEqual({
      sku_code: 'L-1', barcode: undefined, price_minor_units: 200, sale_price_minor_units: null, sale_end_date: null,
      currency_code: 'EUR', is_virtual: false, weight_grams: 5, length_mm: undefined, width_mm: undefined, height_mm: undefined,
      unit_type: 'box', portioned_from_sku_id: null, scale_code: null, pricing_mode: 'fixed', price_per_kilo_minor_units: null, status: 'active',
    })
    expect(p).not.toHaveProperty('tax_class_id')
  })

  it('sends the kilo price only for per-weight pricing', () => {
    const d = { ...emptyVariantSkuDraft('EUR'), pricePerKilo: '9.5' }
    expect(variantSkuAddPayload(d).price_per_kilo_minor_units).toBeNull()
    expect(variantSkuAddPayload({ ...d, pricingMode: 'per_weight' }).price_per_kilo_minor_units).toBe(950)
  })
})

describe('variant edit form', () => {
  it('prefills from the variant and its SKU', () => {
    expect(variantEditFormFrom(variant, sku)).toEqual({
      name: 'Large', skuCode: 'A-1', price: '19.99', salePrice: '15.00', saleEndDate: '', currency: 'EUR', status: 'inactive',
      isVirtual: false, weightGrams: '300', lengthMM: '', widthMM: '', heightMM: '',
    })
  })

  it('prefills just the name for a variant without a SKU', () => {
    expect(variantEditFormFrom(variant, undefined).name).toBe('Large')
  })

  it('saves edited fields, passing barcode, pool and scale fields through untouched (no tax class)', () => {
    const f = { ...variantEditFormFrom(variant, sku), skuCode: 'A-2', price: '25', isVirtual: true }
    const p = variantEditSkuPayload(f, sku)
    expect(p).toMatchObject({
      sku_code: 'A-2', price_minor_units: 2500, sale_price_minor_units: 1500, is_virtual: true, weight_grams: undefined, status: 'inactive',
      barcode: '123', tracks_expiry: true, unit_type: 'kilo', scale_code: '2900633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 2999,
    })
    expect(p).not.toHaveProperty('tax_class_id')
  })
})
