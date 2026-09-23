import { describe, it, expect } from 'vitest'
import { PICKABLE_STATUSES, quantityToPick } from '../pickable'
import type { SalesOrderLineItem } from '../types'

describe('pickable', () => {
  it('lists the pickable statuses', () => {
    expect(PICKABLE_STATUSES).toEqual(['pending', 'processing', 'on-hold'])
  })

  it('subtracts refunded quantity', () => {
    expect(quantityToPick({ quantity: 3, refunded_quantity: 1 } as SalesOrderLineItem)).toBe(2)
    expect(quantityToPick({ quantity: 3 } as SalesOrderLineItem)).toBe(3)
  })
})
