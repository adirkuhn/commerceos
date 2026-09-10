import { request } from './client'
import type { StorefrontCategory } from '../types'

export function listCategories(): Promise<StorefrontCategory[]> {
  return request<StorefrontCategory[]>('/categories')
}
