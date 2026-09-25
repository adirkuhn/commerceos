import type { SalesOrder, SalesOrderStatus, Address } from './types'

export const STATUS_OPTIONS: SalesOrderStatus[] = ['pending', 'processing', 'on-hold', 'shipped', 'completed', 'cancelled', 'refunded', 'failed', 'partially_refunded']

// customerName is the billing name, else the billing email, else
// unknownLabel (admin tables use an em dash; scanner screens spell it out).
export function customerName(order: SalesOrder, unknownLabel = '—'): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || unknownLabel
}

// shippingAddressLine is a one-line "street, postcode city, country"
// summary for queue rows and headers.
export function shippingAddressLine(address?: Address): string {
  if (!address) return ''
  const streetAndLocality = [address.address_1, address.address_2, [address.postcode, address.city].filter(Boolean).join(' ')]
    .filter(Boolean).join(', ')
  return [streetAndLocality, address.country].filter(Boolean).join(', ')
}

// Billing shown as "Same as shipping address" when the two addresses match
// on every field a customer would actually check -- not e.g. phone, which
// legitimately differs (billing contact vs. delivery contact) without the
// addresses being different.
export function addressesEqual(a?: Address, b?: Address): boolean {
  if (!a || !b) return false
  const fields: (keyof Address)[] = ['first_name', 'last_name', 'company', 'address_1', 'address_2', 'city', 'state', 'postcode', 'country']
  return fields.every(f => (a[f] ?? '') === (b[f] ?? ''))
}
