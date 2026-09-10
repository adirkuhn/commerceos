import { useQuery } from '@tanstack/react-query'
import { listActiveFlashSales } from '@/api/promotions'

export function useActiveFlashSales() {
  return useQuery({ queryKey: ['flash-sales'], queryFn: listActiveFlashSales, staleTime: 60 * 1000 })
}
