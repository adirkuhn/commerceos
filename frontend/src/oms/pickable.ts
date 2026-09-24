import type { SalesOrderLineItem } from './types'

// What's still to fetch for a line: ordered minus refunded.
export function quantityToPick(li: SalesOrderLineItem): number {
  return li.quantity - (li.refunded_quantity ?? 0)
}
