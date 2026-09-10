import { useQuery } from '@tanstack/react-query'
import { listProducts, getProductBySlug, getProductsByIds } from '@/api/products'
import type { ProductListParams } from '@/repositories'

export function useProducts(params: ProductListParams) {
  return useQuery({ queryKey: ['products', params], queryFn: () => listProducts(params) })
}

export function useProduct(slug: string | undefined) {
  return useQuery({
    queryKey: ['product', slug],
    queryFn: () => getProductBySlug(slug!),
    enabled: !!slug,
  })
}

export function useProductsByIds(ids: string[]) {
  return useQuery({
    queryKey: ['products-by-ids', ids],
    queryFn: () => getProductsByIds(ids),
    enabled: ids.length > 0,
  })
}
