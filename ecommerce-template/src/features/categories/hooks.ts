import { useQuery } from '@tanstack/react-query'
import { listCategories, getCategoryBySlug, listBrands } from '@/api/categories'

export function useCategories() {
  return useQuery({ queryKey: ['categories'], queryFn: listCategories, staleTime: 5 * 60 * 1000 })
}

export function useCategory(slug: string | undefined) {
  return useQuery({ queryKey: ['category', slug], queryFn: () => getCategoryBySlug(slug!), enabled: !!slug })
}

export function useBrands() {
  return useQuery({ queryKey: ['brands'], queryFn: listBrands, staleTime: 5 * 60 * 1000 })
}
