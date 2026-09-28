import { describe, it, expect } from 'vitest'
import { complete, deliverable, emptyAddress, safePaymentURL } from '../address'

describe('checkout address helpers', () => {
  it('knows when delivery can be priced and when the order can be placed', () => {
    const a = { ...emptyAddress(), address1: 'Main 1', city: 'Utrecht', postcode: '3511 AA' }
    expect(deliverable(a)).toBe(true)
    expect(complete(a)).toBe(false)
    expect(complete({ ...a, first_name: 'Ann', last_name: 'B' })).toBe(true)
  })

  it('only follows https payment pages', () => {
    expect(safePaymentURL('https://checkout.stripe.com/x')).toBe('https://checkout.stripe.com/x')
    expect(safePaymentURL('javascript:alert(1)')).toBeUndefined()
    expect(safePaymentURL('http://x.test')).toBeUndefined()
    expect(safePaymentURL('not a url')).toBeUndefined()
    expect(safePaymentURL(null)).toBeUndefined()
  })
})
