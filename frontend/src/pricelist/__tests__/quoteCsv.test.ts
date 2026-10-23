import { describe, it, expect } from 'vitest'
import { quoteCsv, productLabel } from '../quoteCsv'

const headers = ['Product', 'Quantity', 'Unit price'] as const

describe('quoteCsv', () => {
  it('uses comma and decimal point for locales that write 1.5', () => {
    const csv = quoteCsv([{ product: 'Feijão 1kg (FEI-1)', quantity: 100, unitPriceMinor: 899 }], headers, 'en-US')
    expect(csv).toBe('Product,Quantity,Unit price\r\nFeijão 1kg (FEI-1),100,8.99\r\n')
  })

  it('uses semicolon and decimal comma where Excel expects them (e.g. nl)', () => {
    const csv = quoteCsv([{ product: 'Farofa', quantity: 3, unitPriceMinor: 1250 }], headers, 'nl-NL')
    expect(csv).toBe('Product;Quantity;Unit price\r\nFarofa;3;12,50\r\n')
  })

  it('quotes fields holding the separator, quotes or line breaks', () => {
    const csv = quoteCsv([{ product: 'Pão "de queijo", 1kg', quantity: 1, unitPriceMinor: 100 }], headers, 'en-US')
    expect(csv.split('\r\n')[1]).toBe('"Pão ""de queijo"", 1kg",1,1.00')
  })
})

describe('productLabel', () => {
  it('names product, variant and SKU code', () => {
    expect(productLabel({ product_name: 'Feijão', variant_name: '1kg', sku_code: 'FEI-1' })).toBe('Feijão — 1kg (FEI-1)')
    expect(productLabel({ product_name: 'Farofa', variant_name: '', sku_code: 'FAR' })).toBe('Farofa (FAR)')
  })
})
