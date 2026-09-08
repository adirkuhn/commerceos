import { request } from './client'
import type { StorefrontProduct, StorefrontProductListResult } from '../types'

export interface ListProductsParams {
  categoryId?: string
  search?: string
  page?: number
  perPage?: number
}

export function listProducts(params: ListProductsParams = {}): Promise<StorefrontProductListResult> {
  const query = new URLSearchParams()
  if (params.categoryId) query.set('category_id', params.categoryId)
  if (params.search) query.set('search', params.search)
  if (params.page) query.set('page', String(params.page))
  if (params.perPage) query.set('per_page', String(params.perPage))
  const qs = query.toString()
  return request<StorefrontProductListResult>(`/products${qs ? `?${qs}` : ''}`)
}

export function getProduct(id: string): Promise<StorefrontProduct> {
  return request<StorefrontProduct>(`/products/${encodeURIComponent(id)}`)
}
