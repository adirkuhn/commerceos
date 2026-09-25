import { describe, it, expect } from 'vitest'
import { filterPickingQueue } from '../pickingQueueFilter'
import type { QueueEntry, SalesOrder } from '../../oms/types'

function entry(overrides: Partial<SalesOrder>): QueueEntry {
  const order = {
    id: 'so', external_order_id: 'EXT-9', number: '', billing: { first_name: 'Jane', last_name: 'Doe' }, ...overrides,
  } as SalesOrder
  return { sales_order: order, lock: null, in_packing: false }
}

const entries = [entry({ id: '1', number: '1001' }), entry({ id: '2', billing: { first_name: 'Bob', last_name: 'Smith' } })]

describe('filterPickingQueue', () => {
  it('returns everything for a blank search', () => {
    expect(filterPickingQueue(entries, '  ')).toHaveLength(2)
  })

  it('matches the order number', () => {
    expect(filterPickingQueue(entries, '1001').map(e => e.sales_order.id)).toEqual(['1'])
  })

  it('falls back to the external id when there is no number', () => {
    expect(filterPickingQueue(entries, 'ext-9').map(e => e.sales_order.id)).toEqual(['2'])
  })

  it('matches the customer name case-insensitively', () => {
    expect(filterPickingQueue(entries, 'bob').map(e => e.sales_order.id)).toEqual(['2'])
  })
})
