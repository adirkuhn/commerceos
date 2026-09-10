import type { Cart, Order, ShippingMethodId } from '@/types'
import type { CheckoutInput, CheckoutRepository } from '../types'
import { simulateLatency } from './latency'
import { ORDERS } from '@/config/demoData'

const SHIPPING_PRICE: Record<ShippingMethodId, number> = { standard: 900, express: 2400, pickup: 0 }

let orderSeq = 20000

export class MockCheckoutRepository implements CheckoutRepository {
  async placeOrder(cart: Cart, input: CheckoutInput): Promise<Order> {
    await simulateLatency()
    if (cart.items.length === 0) throw new Error('Your cart is empty.')
    orderSeq += 1
    const shipping = cart.totals.shipping > 0 ? cart.totals.shipping : SHIPPING_PRICE[input.shippingMethodId]
    const total = cart.totals.subtotal - cart.totals.discount + shipping + cart.totals.tax
    const order: Order = {
      id: `ord-${orderSeq}`,
      orderNumber: `AU-${orderSeq}`,
      customerId: input.customerId,
      guestEmail: input.customerId ? undefined : input.email,
      lines: cart.items.map(i => ({
        productId: i.productId, variantId: i.variantId, name: i.name, variantLabel: i.variantLabel,
        imageUrl: i.imageUrl, sku: i.sku, price: i.price, quantity: i.quantity,
      })),
      subtotal: cart.totals.subtotal,
      discount: cart.totals.discount,
      shipping,
      tax: cart.totals.tax,
      total,
      currency: cart.currency,
      status: 'pending',
      paymentStatus: input.paymentMethodId === 'cod' ? 'pending' : 'paid',
      paymentMethod: input.paymentMethodId,
      shippingMethod: input.shippingMethodId,
      billingAddress: input.billingAddress,
      shippingAddress: input.shippingAddress,
      trackingEvents: [{ status: 'Order placed', occurredAt: new Date().toISOString() }],
      estimatedDeliveryLabel: input.shippingMethodId === 'express' ? 'Arriving in 2 business days' : 'Arriving in 5-7 business days',
      placedAt: new Date().toISOString(),
    }
    ORDERS.unshift(order)
    return order
  }
}
