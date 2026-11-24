import { describe, it, expect } from 'vitest'
import { oneLineAddress, shippingAddressName, toOrderAddress } from '../addressLine'
import type { ShippingAddress } from '../../types'

const dock: ShippingAddress = {
  id: 'sa-2', customer_id: 'c-1', label: 'Dock', first_name: 'Ana', last_name: '', company: '', address_1: 'Dock 2', address_2: '',
  city: 'Utrecht', state: '', postcode: '3511 AB', country: 'NL', phone: '06', is_default: false,
}

describe('addressLine', () => {
  it('joins the parts an address has', () => {
    expect(oneLineAddress(dock)).toBe('Dock 2, 3511 AB Utrecht, NL')
    expect(oneLineAddress({ city: 'Delft' })).toBe('Delft')
  })

  it('names an address by its label, else its line', () => {
    expect(shippingAddressName(dock)).toBe('Dock')
    expect(shippingAddressName({ ...dock, label: '' })).toBe('Dock 2, 3511 AB Utrecht, NL')
  })

  it('copies a book address into an order address', () => {
    expect(toOrderAddress(dock)).toEqual({
      first_name: 'Ana', last_name: '', company: '', address_1: 'Dock 2', address_2: '', city: 'Utrecht', state: '', postcode: '3511 AB', country: 'NL', phone: '06',
    })
  })
})
