import i18next from 'i18next'
import type { SalesOrder } from '../oms/types'
import { addressesEqual, shippingAddressLine, shippingMethodLabel } from '../oms/salesOrderDisplay'

// Billing address, shown only when it differs from where the order ships.
export function PickingBillingLine({ order }: { order: SalesOrder }) {
  if (!shippingAddressLine(order.billing) || addressesEqual(order.billing, order.shipping)) return null
  return <p className="text-sm text-amber-400 mt-0.5">{i18next.t('scanner:session.billing', { address: shippingAddressLine(order.billing) })}</p>
}

export function PickingShippingMethodLine({ order }: { order: SalesOrder }) {
  const method = shippingMethodLabel(order)
  if (!method) return null
  return <p className="text-sm text-gray-400 mt-0.5">{i18next.t('scanner:session.shipping', { method })}</p>
}
