import type { Address } from './customer'

export type OrderStatus = 'pending' | 'processing' | 'shipped' | 'delivered' | 'cancelled' | 'refunded'
export type PaymentStatus = 'pending' | 'paid' | 'failed' | 'refunded'

export interface OrderLine {
  productId: string
  variantId?: string
  name: string
  variantLabel?: string
  imageUrl?: string
  sku: string
  price: number
  quantity: number
}

export type ShippingMethodId = 'standard' | 'express' | 'pickup'

export interface ShippingMethod {
  id: ShippingMethodId
  label: string
  description: string
  price: number
  etaLabel: string
}

export type PaymentMethodId = 'card' | 'paypal' | 'apple_pay' | 'google_pay' | 'bank_transfer' | 'cod' | 'bnpl'

export interface PaymentMethodOption {
  id: PaymentMethodId
  label: string
  description: string
  icon: string
}

export interface TrackingEvent {
  status: string
  location?: string
  occurredAt: string
}

export interface ReturnRequest {
  id: string
  orderId: string
  lineProductIds: string[]
  reason: string
  status: 'requested' | 'approved' | 'rejected' | 'refunded'
  refundStatus: 'pending' | 'issued' | 'not_applicable'
  requestedAt: string
}

export interface Order {
  id: string
  orderNumber: string
  customerId?: string
  guestEmail?: string
  lines: OrderLine[]
  subtotal: number
  discount: number
  shipping: number
  tax: number
  total: number
  currency: string
  status: OrderStatus
  paymentStatus: PaymentStatus
  paymentMethod: PaymentMethodId
  shippingMethod: ShippingMethodId
  billingAddress: Address
  shippingAddress: Address
  trackingNumber?: string
  trackingEvents: TrackingEvent[]
  estimatedDeliveryLabel?: string
  placedAt: string
}
