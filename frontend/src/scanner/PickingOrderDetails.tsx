import type { SalesOrder } from '../oms/types'
import { addressesEqual, shippingAddressLine } from '../oms/salesOrderDisplay'

// Billing address, shown only when it differs from where the order ships.
export function PickingBillingLine({ order }: { order: SalesOrder }) {
  if (!shippingAddressLine(order.billing) || addressesEqual(order.billing, order.shipping)) return null
  return <p className="text-sm text-amber-400 mt-0.5">Billing: {shippingAddressLine(order.billing)}</p>
}

export function PickingShippingMethodLine({ order }: { order: SalesOrder }) {
  if (!(order.shipping_method_title || order.shipping_method)) return null
  return <p className="text-sm text-gray-400 mt-0.5">Shipping: {order.shipping_method_title || order.shipping_method}</p>
}
