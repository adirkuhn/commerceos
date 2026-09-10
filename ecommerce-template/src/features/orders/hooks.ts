import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listOrdersForCustomer, getOrderById, cancelOrder, requestOrderReturn } from '@/api/orders'

export function useCustomerOrders(customerId: string | undefined) {
  return useQuery({ queryKey: ['orders', customerId], queryFn: () => listOrdersForCustomer(customerId!), enabled: !!customerId })
}

export function useOrder(id: string | undefined) {
  return useQuery({ queryKey: ['order', id], queryFn: () => getOrderById(id!), enabled: !!id })
}

export function useCancelOrder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => cancelOrder(id),
    onSuccess: (order) => {
      queryClient.setQueryData(['order', order.id], order)
      queryClient.invalidateQueries({ queryKey: ['orders'] })
    },
  })
}

export function useRequestReturn() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ orderId, lineProductIds, reason }: { orderId: string; lineProductIds: string[]; reason: string }) =>
      requestOrderReturn(orderId, lineProductIds, reason),
    onSuccess: (order) => queryClient.invalidateQueries({ queryKey: ['order', order.id] }),
  })
}
