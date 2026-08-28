import { describe, it, expect } from 'vitest'
import { guessCarrierFromShippingMethod } from '../carrierGuess'

describe('guessCarrierFromShippingMethod', () => {
  it('matches PostNL by title', () => {
    expect(guessCarrierFromShippingMethod('PostNL PakjeGemak', 'flat_rate', 99)).toBe(1)
  })

  it('matches bpost by title, case-insensitively', () => {
    expect(guessCarrierFromShippingMethod('BPOST home delivery', 'flat_rate', 99)).toBe(2)
  })

  it('matches DPD by title', () => {
    expect(guessCarrierFromShippingMethod('DPD Pickup', 'flat_rate', 99)).toBe(4)
  })

  it('matches DHL by title', () => {
    expect(guessCarrierFromShippingMethod('DHL Europe', 'flat_rate', 99)).toBe(9)
  })

  it('matches UPS by title', () => {
    expect(guessCarrierFromShippingMethod('UPS Standard', 'flat_rate', 99)).toBe(12)
  })

  it('falls back to the given default when the title names no known carrier', () => {
    expect(guessCarrierFromShippingMethod('Flat rate', 'flat_rate', 99)).toBe(99)
  })

  it('falls back when both title and method are missing', () => {
    expect(guessCarrierFromShippingMethod(undefined, undefined, 99)).toBe(99)
  })

  it('falls back to the machine slug when the title is empty but the slug names a carrier', () => {
    expect(guessCarrierFromShippingMethod('', 'postnl_shipping', 99)).toBe(1)
  })
})
