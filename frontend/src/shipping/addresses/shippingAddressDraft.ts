import type { ShippingAddress } from '../types'

// The editable form of a ShippingAddress: every field a string.
export type AddressDraft = Required<ShippingAddress>
export type AddressField = keyof AddressDraft

export const EMPTY_ADDRESS_DRAFT: AddressDraft = {
  company: '', street: '', number: '', number_suffix: '', postal_code: '', city: '', cc: '', email: '', phone: '',
}

// Mirrors the backend ShippingAddress schema (openapi maxLength / required).
const MAX_LENGTH: Record<AddressField, number> = {
  company: 35, street: 95, number: 10, number_suffix: 10, postal_code: 17, city: 35, cc: 2, email: 50, phone: 17,
}
const REQUIRED: readonly AddressField[] = ['company', 'street', 'number', 'postal_code', 'city', 'cc']
const OPTIONAL: readonly AddressField[] = ['number_suffix', 'email', 'phone']
const COUNTRY_CODE = /^[A-Za-z]{2}$/

// Why a field is invalid -- an i18n key under shipping:addresses.errors.
export type AddressFieldError = 'required' | 'country' | 'tooLong'
export type AddressErrors = Partial<Record<AddressField, AddressFieldError>>

export function draftOfAddress(address: ShippingAddress | null): AddressDraft {
  return { ...EMPTY_ADDRESS_DRAFT, ...address }
}

// addressOfDraft is what is sent: trimmed, country upper-cased, blank
// optional fields left out.
export function addressOfDraft(draft: AddressDraft): ShippingAddress {
  const address: ShippingAddress = {
    company: draft.company.trim(), street: draft.street.trim(), number: draft.number.trim(),
    postal_code: draft.postal_code.trim(), city: draft.city.trim(), cc: draft.cc.trim().toUpperCase(),
  }
  for (const field of OPTIONAL) {
    const value = draft[field].trim()
    if (value) address[field] = value
  }
  return address
}

// validateAddressDraft checks what the backend would reject, so the form
// can point at the field before saving. Empty = valid.
export function validateAddressDraft(draft: AddressDraft): AddressErrors {
  const errors: AddressErrors = {}
  for (const field of Object.keys(MAX_LENGTH) as AddressField[]) {
    const value = draft[field].trim()
    if (REQUIRED.includes(field) && !value) errors[field] = 'required'
    else if (value.length > MAX_LENGTH[field]) errors[field] = 'tooLong'
  }
  if (errors.cc !== 'required' && !COUNTRY_CODE.test(draft.cc.trim())) errors.cc = 'country'
  return errors
}
