import { request } from './client'
import type { StorefrontOrderStatus } from '../types'

export function lookupOrder(orderNumber: string, email: string): Promise<StorefrontOrderStatus> {
  const query = new URLSearchParams({ order_number: orderNumber, email })
  return request<StorefrontOrderStatus>(`/orders/lookup?${query.toString()}`)
}
