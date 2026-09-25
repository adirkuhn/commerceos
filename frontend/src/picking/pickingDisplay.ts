import type { SalesOrder } from '../oms/types'
import type { PickScan } from '../scanner/types'
import type { Address } from '../oms/types'

export function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

export function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

export function shippingAddressLine(address?: Address): string {
  if (!address) return ''
  const streetAndLocality = [address.address_1, address.address_2, [address.postcode, address.city].filter(Boolean).join(' ')]
    .filter(Boolean).join(', ')
  return [streetAndLocality, address.country].filter(Boolean).join(', ')
}

export const ACTION_LABEL: Record<PickScan['action_type'], string> = {
  pick: 'Picked', substitute: 'Substituted', freebie: 'Freebie', new_item: 'New item', refund: 'Refunded',
}

export const PICKING_PATH = '/picking'
