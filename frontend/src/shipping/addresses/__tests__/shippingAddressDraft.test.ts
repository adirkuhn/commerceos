import { describe, it, expect } from 'vitest'
import { EMPTY_ADDRESS_DRAFT, addressOfDraft, draftOfAddress, validateAddressDraft } from '../shippingAddressDraft'

const full = {
  company: ' Shop BV ', street: 'Damrak', number: '1', number_suffix: '', postal_code: '1012 LG', city: 'Amsterdam',
  cc: 'nl', email: '', phone: '+31 20 1234567',
}

describe('draftOfAddress', () => {
  it('fills every field, blank for missing optional ones', () => {
    expect(draftOfAddress({ company: 'A', street: 'B', number: '2', postal_code: 'P', city: 'C', cc: 'BE' }))
      .toEqual({ company: 'A', street: 'B', number: '2', number_suffix: '', postal_code: 'P', city: 'C', cc: 'BE', email: '', phone: '' })
  })

  it('is empty without an address', () => {
    expect(draftOfAddress(null)).toEqual(EMPTY_ADDRESS_DRAFT)
  })
})

describe('addressOfDraft', () => {
  it('trims, upper-cases the country and leaves out blank optional fields', () => {
    expect(addressOfDraft(full)).toEqual({
      company: 'Shop BV', street: 'Damrak', number: '1', postal_code: '1012 LG', city: 'Amsterdam', cc: 'NL', phone: '+31 20 1234567',
    })
  })
})

describe('validateAddressDraft', () => {
  it('accepts a complete address', () => {
    expect(validateAddressDraft(full)).toEqual({})
  })

  it('flags each missing required field', () => {
    expect(Object.keys(validateAddressDraft(EMPTY_ADDRESS_DRAFT)).sort())
      .toEqual(['cc', 'city', 'company', 'number', 'postal_code', 'street'])
  })

  it('flags a country that is not 2 letters and fields that are too long', () => {
    const errors = validateAddressDraft({ ...full, cc: 'NLD', company: 'x'.repeat(36) })
    expect(errors.cc).toBe('country')
    expect(errors.company).toBe('tooLong')
  })
})
