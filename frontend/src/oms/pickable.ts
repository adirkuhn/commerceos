import type { SalesOrderLineItem, SalesOrderStatus } from './types'

// Mirrors backend domain/oms.PickableStatuses -- the statuses an order can
// still be picked in. One place, so the picking queues can't drift.
export const PICKABLE_STATUSES: SalesOrderStatus[] = ['pending', 'processing', 'on-hold']

// What's still to fetch for a line: ordered minus refunded.
export function quantityToPick(li: SalesOrderLineItem): number {
  return li.quantity - (li.refunded_quantity ?? 0)
}
