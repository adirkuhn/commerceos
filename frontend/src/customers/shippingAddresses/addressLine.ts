import type { Address, ShippingAddress } from '../types'

/** One line for an address: street, postcode + city, country -- the parts it has. */
export function oneLineAddress(a: Address | ShippingAddress): string {
  const place = [a.postcode, a.city].filter(Boolean).join(' ')
  return [a.address_1, a.address_2, place, a.country].filter(Boolean).join(', ')
}

/** How an address is named in lists and pickers: its label, else its line. */
export function shippingAddressName(a: ShippingAddress): string {
  return a.label || oneLineAddress(a)
}

/** The order-shaped copy of a book address (no label, id or flag). */
export function toOrderAddress(a: ShippingAddress): Address {
  return {
    first_name: a.first_name, last_name: a.last_name, company: a.company, address_1: a.address_1, address_2: a.address_2,
    city: a.city, state: a.state, postcode: a.postcode, country: a.country, phone: a.phone,
  }
}
