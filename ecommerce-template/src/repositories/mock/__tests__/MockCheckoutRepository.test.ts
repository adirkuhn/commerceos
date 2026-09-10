import { describe, expect, it } from 'vitest'
import { MockCheckoutRepository } from '../MockCheckoutRepository'
import type { Cart, Address } from '@/types'

const repo = new MockCheckoutRepository()

const address: Address = {
  id: 'a1', firstName: 'Morgan', lastName: 'Lee', line1: '1 Main St', city: 'Boulder', postalCode: '80302', country: 'US',
}

function cart(overrides: Partial<Cart> = {}): Cart {
  return {
    id: 'cart-1',
    items: [{ id: 'line-1', productId: 'p1', name: 'Trail Runner', sku: 'SKU-1', price: 14500, quantity: 1 }],
    savedItems: [], coupons: [], currency: 'USD',
    totals: { subtotal: 14500, discount: 0, shipping: 900, tax: 1160, total: 16560, freeShippingThreshold: 10000, freeShippingRemaining: 0 },
    ...overrides,
  }
}

describe('MockCheckoutRepository', () => {
  it('places an order from a cart', async () => {
    const order = await repo.placeOrder(cart(), {
      email: 'morgan@example.com', billingAddress: address, shippingAddress: address, shippingMethodId: 'standard', paymentMethodId: 'card',
    })
    expect(order.lines).toHaveLength(1)
    expect(order.total).toBe(16560)
    expect(order.paymentStatus).toBe('paid')
    expect(order.status).toBe('pending')
  })

  it('marks cash on delivery orders as payment pending, not paid', async () => {
    const order = await repo.placeOrder(cart(), {
      email: 'morgan@example.com', billingAddress: address, shippingAddress: address, shippingMethodId: 'standard', paymentMethodId: 'cod',
    })
    expect(order.paymentStatus).toBe('pending')
  })

  it('rejects placing an order from an empty cart', async () => {
    await expect(repo.placeOrder(cart({ items: [] }), {
      email: 'morgan@example.com', billingAddress: address, shippingAddress: address, shippingMethodId: 'standard', paymentMethodId: 'card',
    })).rejects.toThrow()
  })

  it('assigns a guest email when no customerId is given, and none when there is one', async () => {
    const guestOrder = await repo.placeOrder(cart(), {
      email: 'guest@example.com', billingAddress: address, shippingAddress: address, shippingMethodId: 'standard', paymentMethodId: 'card',
    })
    expect(guestOrder.guestEmail).toBe('guest@example.com')

    const customerOrder = await repo.placeOrder(cart(), {
      email: 'morgan@example.com', billingAddress: address, shippingAddress: address, shippingMethodId: 'standard', paymentMethodId: 'card', customerId: 'cust-demo',
    })
    expect(customerOrder.guestEmail).toBeUndefined()
    expect(customerOrder.customerId).toBe('cust-demo')
  })
})
