import { describe, it, expect } from 'vitest'
import { profitAt, profitExclIncl } from '../profit'

describe('profitAt', () => {
  it('profit, margin (of price) and markup (of cost) in basis points', () => {
    expect(profitAt(450, 210)).toEqual({ price: 450, profit: 240, marginBasisPoints: 5333, markupBasisPoints: 11429 })
  })

  it('a price below cost is a loss', () => {
    expect(profitAt(180, 210)).toEqual({ price: 180, profit: -30, marginBasisPoints: -1667, markupBasisPoints: -1429 })
  })

  it('no margin on a zero price, no markup on a zero cost', () => {
    expect(profitAt(0, 210).marginBasisPoints).toBe(0)
    expect(profitAt(450, 0).markupBasisPoints).toBe(0)
  })
})

describe('profitExclIncl', () => {
  it('a VAT-inclusive stored price: inclusive as is, exclusive with VAT taken off', () => {
    const both = profitExclIncl(450, 210, 900, true)
    expect(both?.inclusive.price).toBe(450)
    expect(both?.exclusive.price).toBe(413) // 450 / 1.09
    expect(both?.exclusive.profit).toBe(203)
  })

  it('a VAT-exclusive stored price: exclusive as is, inclusive with VAT added', () => {
    const both = profitExclIncl(450, 210, 2100, false)
    expect(both?.exclusive.price).toBe(450)
    expect(both?.inclusive.price).toBe(545) // 450 * 1.21
  })

  it('is null without a known VAT rate', () => {
    expect(profitExclIncl(450, 210, null, true)).toBeNull()
  })
})
