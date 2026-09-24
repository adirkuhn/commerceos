import { describe, it, expect } from 'vitest'
import { quantityToPick } from '../pickable'
import type { SalesOrderLineItem } from '../types'

describe('pickable', () => {
  it('subtracts refunded quantity', () => {
    expect(quantityToPick({ quantity: 3, refunded_quantity: 1 } as SalesOrderLineItem)).toBe(2)
    expect(quantityToPick({ quantity: 3 } as SalesOrderLineItem)).toBe(3)
  })
})
