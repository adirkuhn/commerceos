import { request } from './client'
import type { StorefrontProductSummary } from '../types'

export function getProductRecommendations(productId: string): Promise<StorefrontProductSummary[]> {
  return request<StorefrontProductSummary[]>(`/products/${encodeURIComponent(productId)}/recommendations`)
}
