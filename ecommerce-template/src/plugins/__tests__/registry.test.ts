import { describe, expect, it } from 'vitest'
import { pluginRegistry } from '../registry'
import type { EcommercePlugin } from '@/types'

// pluginRegistry is a module-level singleton -- clear it via a fresh
// instance import isn't possible, so each test registers its own
// uniquely-named plugin instead of relying on a reset.
function uniquePlugin(overrides: Partial<EcommercePlugin>): EcommercePlugin {
  return { name: `test-plugin-${Math.random()}`, ...overrides }
}

describe('pluginRegistry', () => {
  it('registers a plugin and rejects a duplicate name', () => {
    const plugin = uniquePlugin({})
    pluginRegistry.register(plugin)
    expect(() => pluginRegistry.register(plugin)).toThrow(/already registered/)
  })

  it('collects productTabs across all registered plugins', () => {
    const tab = { id: 'tab-1', label: 'Custom tab', render: () => () => null }
    pluginRegistry.register(uniquePlugin({ productTabs: [tab] }))
    expect(pluginRegistry.productTabs().some(t => t.id === 'tab-1')).toBe(true)
  })

  it('filters checkoutExtensions by step', () => {
    pluginRegistry.register(uniquePlugin({
      checkoutExtensions: [{ id: 'ext-shipping', step: 'shipping', render: () => null }, { id: 'ext-payment', step: 'payment', render: () => null }],
    }))
    const shippingExtensions = pluginRegistry.checkoutExtensions('shipping')
    expect(shippingExtensions.some(e => e.id === 'ext-shipping')).toBe(true)
    expect(shippingExtensions.some(e => e.id === 'ext-payment')).toBe(false)
  })

  it('filters navigationItems by position', () => {
    pluginRegistry.register(uniquePlugin({ navigationItems: [{ label: 'Loyalty', href: '/loyalty', position: 'footer' }] }))
    expect(pluginRegistry.navigationItems('footer').some(n => n.label === 'Loyalty')).toBe(true)
    expect(pluginRegistry.navigationItems('header').some(n => n.label === 'Loyalty')).toBe(false)
  })

  it('invokes every registered hook handler for a given hook name', () => {
    let received: unknown = null
    pluginRegistry.register(uniquePlugin({ hooks: { order_placed: (payload) => { received = payload } } }))
    pluginRegistry.emit('order_placed', { orderId: 'ord-1' })
    expect(received).toEqual({ orderId: 'ord-1' })
  })

  it('does not throw emitting a hook no plugin subscribes to', () => {
    expect(() => pluginRegistry.emit('cart_item_added', {})).not.toThrow()
  })
})
