import { request } from './client'
import type { Cart, CheckoutRequest, Checkout, UpdateCartItemLine } from '../types'

export function createCart(): Promise<Cart> {
  return request<Cart>('/carts', { method: 'POST' })
}

export function getCart(token: string): Promise<Cart> {
  return request<Cart>(`/carts/${encodeURIComponent(token)}`)
}

export function updateCartItems(token: string, items: UpdateCartItemLine[]): Promise<Cart> {
  return request<Cart>(`/carts/${encodeURIComponent(token)}/items`, {
    method: 'PUT',
    body: JSON.stringify({ items }),
  })
}

export function checkoutCart(token: string, payload: CheckoutRequest): Promise<Checkout> {
  return request<Checkout>(`/carts/${encodeURIComponent(token)}/checkout`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}
