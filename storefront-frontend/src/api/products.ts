import { request } from './client'
import type { StorefrontProduct, StorefrontProductListResult } from '../types'

export interface ListProductsParams {
  categoryId?: string
  // productIds narrows to exactly this list (a merchant's manually-curated
  // Featured Products block), on top of channel enrollment. excludeIds
  // removes these regardless of every other filter -- combinable with
  // either categoryId or productIds. See backend
  // application/storefront.ListProductsInput's own doc comments.
  productIds?: string[]
  excludeIds?: string[]
  search?: string
  page?: number
  perPage?: number
}

export function listProducts(params: ListProductsParams = {}): Promise<StorefrontProductListResult> {
  const query = new URLSearchParams()
  if (params.categoryId) query.set('category_id', params.categoryId)
  if (params.productIds && params.productIds.length > 0) query.set('product_ids', params.productIds.join(','))
  if (params.excludeIds && params.excludeIds.length > 0) query.set('exclude_product_ids', params.excludeIds.join(','))
  if (params.search) query.set('search', params.search)
  if (params.page) query.set('page', String(params.page))
  if (params.perPage) query.set('per_page', String(params.perPage))
  const qs = query.toString()
  return request<StorefrontProductListResult>(`/products${qs ? `?${qs}` : ''}`)
}

export function getProduct(id: string): Promise<StorefrontProduct> {
  return request<StorefrontProduct>(`/products/${encodeURIComponent(id)}`)
}
