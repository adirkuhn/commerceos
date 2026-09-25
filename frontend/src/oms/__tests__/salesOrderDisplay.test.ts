import { describe, it, expect } from 'vitest'
import { STATUS_OPTIONS, addressesEqual, customerName, shippingAddressLine } from '../salesOrderDisplay'
import type { SalesOrder } from '../types'
import omsContract from '../../../../backend/api/oms/openapi.yaml?raw'

const order = (billing: SalesOrder['billing']) => ({ billing }) as SalesOrder

describe('customerName', () => {
  it('uses the billing name, then email', () => {
    expect(customerName(order({ first_name: 'Ana', last_name: 'Silva' }))).toBe('Ana Silva')
    expect(customerName(order({ email: 'a@x.com' }))).toBe('a@x.com')
  })
  it('renders an em dash when unknown, or the given fallback', () => {
    expect(customerName(order({}))).toBe('—')
    expect(customerName(order({}), 'Unknown customer')).toBe('Unknown customer')
  })
})

describe('shippingAddressLine', () => {
  it('joins street, postcode+city and country', () => {
    expect(shippingAddressLine({ address_1: 'Main 1', address_2: 'Apt 2', postcode: '1011', city: 'Amsterdam', country: 'NL' }))
      .toBe('Main 1, Apt 2, 1011 Amsterdam, NL')
  })
  it('skips missing parts and handles no address', () => {
    expect(shippingAddressLine({ city: 'Amsterdam' })).toBe('Amsterdam')
    expect(shippingAddressLine(undefined)).toBe('')
  })
})

describe('addressesEqual', () => {
  it('compares every address field, treating missing as empty', () => {
    expect(addressesEqual({ city: 'A', company: '' }, { city: 'A' })).toBe(true)
    expect(addressesEqual({ city: 'A' }, { city: 'B' })).toBe(false)
  })
  it('is false when either side is missing', () => {
    expect(addressesEqual(undefined, { city: 'A' })).toBe(false)
  })
})

// The status list must match the OpenAPI contract's SalesOrderStatus enum
// exactly -- a status missing here renders unstyled and, in the detail
// page's status <select>, silently displays as the first option instead.
describe('STATUS_OPTIONS', () => {
  it('matches the SalesOrderStatus enum in the oms OpenAPI contract', () => {
    const block = omsContract.slice(omsContract.indexOf('\n    SalesOrderStatus:'))
    const enumLine = block.match(/enum: \[([^\]]+)\]/)
    expect(enumLine).not.toBeNull()
    const contract = enumLine![1].split(',').map((s: string) => s.trim())
    expect([...STATUS_OPTIONS].sort()).toEqual([...contract].sort())
  })
})
