import { orderRepository } from '@/repositories'

export const listOrdersForCustomer = (customerId: string) => orderRepository.listForCustomer(customerId)
export const getOrderById = (id: string) => orderRepository.getById(id)
export const cancelOrder = (id: string) => orderRepository.cancel(id)
export const requestOrderReturn = (orderId: string, lineProductIds: string[], reason: string) =>
  orderRepository.requestReturn(orderId, lineProductIds, reason)
