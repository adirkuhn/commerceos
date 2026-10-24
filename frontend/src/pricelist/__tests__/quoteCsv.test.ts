import { describe, it, expect } from 'vitest'
import { quoteCsv, productName } from '../quoteCsv'

const headers = { sku: 'SKU', product: 'Product', quantity: 'Quantity', unitPrice: 'Unit price', lineTotal: 'Line total', total: 'Order total' }

describe('quoteCsv', () => {
  it('writes sku, product, quantity, unit price and line total, then the order total', () => {
    const csv = quoteCsv([
      { sku: 'FEI-1', product: 'Feijão', quantity: 100, unitPriceMinor: 899 },
      { sku: 'FAR-1', product: 'Farofa', quantity: 2, unitPriceMinor: 1250 },
    ], headers, 'en-US')
    expect(csv).toBe(
      'SKU,Product,Quantity,Unit price,Line total\r\n' +
      'FEI-1,Feijão,100,8.99,899.00\r\n' +
      'FAR-1,Farofa,2,12.50,25.00\r\n' +
      ',,,Order total,924.00\r\n')
  })

  it('uses semicolon and decimal comma where Excel expects them (e.g. nl)', () => {
    const csv = quoteCsv([{ sku: 'FAR-1', product: 'Farofa', quantity: 3, unitPriceMinor: 1250 }], headers, 'nl-NL')
    expect(csv.split('\r\n')[1]).toBe('FAR-1;Farofa;3;12,50;37,50')
  })

  it('quotes fields holding the separator, quotes or line breaks', () => {
    const csv = quoteCsv([{ sku: 'P', product: 'Pão "de queijo", 1kg', quantity: 1, unitPriceMinor: 100 }], headers, 'en-US')
    expect(csv.split('\r\n')[1]).toBe('P,"Pão ""de queijo"", 1kg",1,1.00,1.00')
  })
})

describe('productName', () => {
  it('is the product name, with the variant only when it adds something', () => {
    expect(productName({ product_name: 'Paçoca 300g', variant_name: 'Paçoca 300g' })).toBe('Paçoca 300g')
    expect(productName({ product_name: 'Paçoca', variant_name: '' })).toBe('Paçoca')
    expect(productName({ product_name: 'Feijão', variant_name: '1kg' })).toBe('Feijão — 1kg')
  })
})
