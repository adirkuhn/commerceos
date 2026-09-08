import { request } from './client'
import type { Wishlist } from '../types'

export function createWishlist(): Promise<Wishlist> {
  return request<Wishlist>('/wishlists', { method: 'POST' })
}

export function getWishlist(token: string): Promise<Wishlist> {
  return request<Wishlist>(`/wishlists/${encodeURIComponent(token)}`)
}

export function toggleWishlistItem(token: string, productId: string): Promise<Wishlist> {
  return request<Wishlist>(`/wishlists/${encodeURIComponent(token)}/items`, {
    method: 'PUT',
    body: JSON.stringify({ product_id: productId }),
  })
}
