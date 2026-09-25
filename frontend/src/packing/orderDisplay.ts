import type { SalesOrder } from '../oms/types'

export function orderNumber(order: SalesOrder): string {
  return order.number || order.external_order_id
}
