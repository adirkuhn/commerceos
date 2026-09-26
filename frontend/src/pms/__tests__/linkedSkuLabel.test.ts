import { describe, it, expect } from 'vitest'
import { linkedSkuLabel } from '../linkedSkuLabel'

describe('linkedSkuLabel', () => {
  it('names the product and its EAN', () => {
    expect(linkedSkuLabel({ product_name: 'Refresco em pó Uva - MID - Ajinomoto', sku_code: 'REF-UVA', barcode: '7891132082346' }))
      .toBe('Refresco em pó Uva - MID - Ajinomoto · EAN 7891132082346')
  })
  it('falls back to the SKU code without a barcode', () => {
    expect(linkedSkuLabel({ product_name: 'Rapadurinha', sku_code: 'RAPA-25G', barcode: '' })).toBe('Rapadurinha · RAPA-25G')
    expect(linkedSkuLabel({ product_name: 'Rapadurinha', sku_code: 'RAPA-25G', barcode: null })).toBe('Rapadurinha · RAPA-25G')
  })
})
