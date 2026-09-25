import type { SalesOrder, SalesOrderStatus, Address } from './types'

export const STATUS_OPTIONS: SalesOrderStatus[] = ['pending', 'processing', 'on-hold', 'shipped', 'completed', 'cancelled', 'refunded', 'failed']

export function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString()
}

export function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
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
