import type { QueueEntry } from '../oms/types'
import { customerName } from '../oms/salesOrderDisplay'

// The picking queue's search box: matches order number (or the external id
// when there's no number) and customer name, case-insensitively.
export function filterPickingQueue(entries: QueueEntry[], search: string): QueueEntry[] {
  return entries.filter(({ sales_order: o }) => {
    if (!search.trim()) return true
    const q = search.toLowerCase()
    return (o.number || o.external_order_id).toLowerCase().includes(q) || customerName(o).toLowerCase().includes(q)
  })
}
