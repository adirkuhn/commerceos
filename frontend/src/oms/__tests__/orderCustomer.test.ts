import { describe, it, expect } from 'vitest'
import { orderCustomerChange } from '../orderCustomer'

describe('orderCustomerChange', () => {
  it('links the customer and copies the addresses it has', () => {
    expect(orderCustomerChange({ id: 'c-1', email: 'a@x.nl', billing: { first_name: 'Ana', city: 'Utrecht' }, shipping: {} } as never))
      .toEqual({ customer_id: 'c-1', billing: { first_name: 'Ana', city: 'Utrecht' } })
  })

  it('removing the customer clears the link only', () => {
    expect(orderCustomerChange(null)).toEqual({ customer_id: '' })
  })
})
