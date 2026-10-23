import { describe, it, expect } from 'vitest'
import { decodeQuoteLines, encodeQuoteLines } from '../quoteLines'

const P = '0b0e7e8e-0000-4000-8000-000000000001'
const S = '0b0e7e8e-0000-4000-8000-0000000000a1'

describe('quote lines in the URL', () => {
  it('round-trips lines, with and without a typed price', () => {
    const lines = [
      { productId: P, skuId: S, quantity: 100 },
      { productId: P, skuId: 's2', quantity: 0, unitPrice: 899 },
    ]
    expect(decodeQuoteLines(encodeQuoteLines(lines))).toEqual(lines)
  })

  it('encodes nothing as null so the param leaves the URL', () => {
    expect(encodeQuoteLines([])).toBeNull()
  })

  it('drops malformed entries instead of rendering broken lines', () => {
    expect(decodeQuoteLines(`${P}.${S}.5,garbage,${P}.${S}.-1,${P}.${S}.2.x`)).toEqual([{ productId: P, skuId: S, quantity: 5 }])
    expect(decodeQuoteLines(null)).toEqual([])
  })
})
