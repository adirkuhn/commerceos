import { productRepository } from '@/repositories'
import type { ProductListParams } from '@/repositories'

export const listProducts = (params: ProductListParams = {}) => productRepository.list(params)
export const getProductBySlug = (slug: string) => productRepository.getBySlug(slug)
export const getProductById = (id: string) => productRepository.getById(id)
export const getProductsByIds = (ids: string[]) => productRepository.getByIds(ids)
