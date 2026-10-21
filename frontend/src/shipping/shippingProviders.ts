import { providerLabel } from '../webhooks/providerLabel'
import type { ShippingConnection } from './types'

// The shipping providers the label flow knows -- the backend's connection
// provider strings.
export const SHIPPING_PROVIDERS = {
  myparcel: 'myparcel',
  postnl: 'postnl',
} as const

// What a provider's label flow offers, so screens ask "does it have X?"
// instead of checking provider names.
export interface ShippingProviderCapabilities {
  // The provider ships through several carriers (MyParcel: PostNL, DPD,
  // DHL, ...) and staff pick one per label. A provider that is its own
  // carrier (PostNL) has no carrier choice.
  hasCarrierChoice: boolean
  // The order is also exported to the provider's own order dashboard
  // before the label is made (MyParcel's Orders API, on the sales order
  // page).
  exportsOrders: boolean
}

const CAPABILITIES: Record<string, ShippingProviderCapabilities> = {
  [SHIPPING_PROVIDERS.myparcel]: { hasCarrierChoice: true, exportsOrders: true },
  [SHIPPING_PROVIDERS.postnl]: { hasCarrierChoice: false, exportsOrders: false },
}

// A provider this frontend doesn't know yet gets the plain label flow.
const UNKNOWN_PROVIDER: ShippingProviderCapabilities = { hasCarrierChoice: false, exportsOrders: false }

export function shippingProviderCapabilities(provider: string | undefined): ShippingProviderCapabilities {
  return (provider && CAPABILITIES[provider]) || UNKNOWN_PROVIDER
}

// The name a picker shows for a connection: the provider's name, plus the
// connection date when the tenant has more than one connection of that
// provider (two MyParcel accounts would read the same otherwise).
export function shippingConnectionLabel(connection: ShippingConnection, all: readonly ShippingConnection[]): string {
  const name = providerLabel(connection.provider)
  const sameProvider = all.filter(c => c.provider === connection.provider).length
  return sameProvider > 1 ? `${name} (${connection.created_at.slice(0, 10)})` : name
}
