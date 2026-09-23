import { describe, it, expect } from 'vitest'
import { customerName, formatMoney, orderNumber } from '../orderDisplay'
import type { SalesOrder } from '../../oms/types'

const base = { id: 'so-1', number: '', external_order_id: 'ext-9' } as SalesOrder

describe('orderDisplay', () => {
  it('orderNumber prefers number, falls back to external id', () => {
    expect(orderNumber({ ...base, number: '1001' })).toBe('1001')
    expect(orderNumber(base)).toBe('ext-9')
  })

  it('customerName joins billing names, falls back to email then a dash', () => {
    expect(customerName({ ...base, billing: { first_name: 'Ann', last_name: 'Lee' } } as SalesOrder)).toBe('Ann Lee')
    expect(customerName({ ...base, billing: { email: 'a@b.c' } } as SalesOrder)).toBe('a@b.c')
    expect(customerName(base)).toBe('—')
  })

  it('formatMoney formats minor units, surviving a bad currency code', () => {
    expect(formatMoney(1250, 'EUR')).toContain('12.50')
    expect(formatMoney(1250, 'NOT-A-CODE')).toBe('12.50 NOT-A-CODE')
  })
})
