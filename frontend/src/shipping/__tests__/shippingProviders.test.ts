import { describe, it, expect } from 'vitest'
import { shippingConnectionLabel, shippingProviderCapabilities, SHIPPING_PROVIDERS } from '../shippingProviders'
import type { ShippingConnection } from '../types'

const conn = (id: string, provider: string, created_at = '2026-01-01T00:00:00Z'): ShippingConnection =>
  ({ id, provider, status: 'active', created_at, is_default: false })

describe('shippingProviderCapabilities', () => {
  it('MyParcel has a carrier choice and exports orders', () => {
    expect(shippingProviderCapabilities(SHIPPING_PROVIDERS.myparcel)).toEqual({ hasCarrierChoice: true, exportsOrders: true })
  })

  it('PostNL is its own carrier', () => {
    expect(shippingProviderCapabilities(SHIPPING_PROVIDERS.postnl)).toEqual({ hasCarrierChoice: false, exportsOrders: false })
  })

  it('an unknown or missing provider gets the plain flow', () => {
    expect(shippingProviderCapabilities('sendcloud').hasCarrierChoice).toBe(false)
    expect(shippingProviderCapabilities(undefined).exportsOrders).toBe(false)
  })
})

describe('shippingConnectionLabel', () => {
  it('is the provider name', () => {
    const all = [conn('a', 'myparcel'), conn('b', 'postnl')]
    expect(shippingConnectionLabel(all[0], all)).toBe('MyParcel')
    expect(shippingConnectionLabel(all[1], all)).toBe('PostNL')
  })

  it('adds the connection date when a provider is connected twice', () => {
    const all = [conn('a', 'myparcel', '2026-01-01T00:00:00Z'), conn('b', 'myparcel', '2026-03-05T10:00:00Z')]
    expect(shippingConnectionLabel(all[1], all)).toBe('MyParcel (2026-03-05)')
  })
})
