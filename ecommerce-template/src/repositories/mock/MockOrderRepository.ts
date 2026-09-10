import type { Order } from '@/types'
import type { OrderRepository } from '../types'
import { simulateLatency } from './latency'
import { ORDERS } from '@/config/demoData'

export class MockOrderRepository implements OrderRepository {
  async listForCustomer(customerId: string): Promise<Order[]> {
    await simulateLatency()
    return ORDERS.filter(o => o.customerId === customerId).sort((a, b) => new Date(b.placedAt).getTime() - new Date(a.placedAt).getTime())
  }

  async getById(id: string): Promise<Order | null> {
    await simulateLatency()
    return ORDERS.find(o => o.id === id) ?? null
  }

  async cancel(id: string): Promise<Order> {
    await simulateLatency()
    const order = ORDERS.find(o => o.id === id)
    if (!order) throw new Error('Order not found.')
    if (order.status !== 'pending' && order.status !== 'processing') {
      throw new Error('This order can no longer be cancelled.')
    }
    order.status = 'cancelled'
    return order
  }

  async requestReturn(orderId: string, lineProductIds: string[], reason: string): Promise<Order> {
    await simulateLatency()
    const order = ORDERS.find(o => o.id === orderId)
    if (!order) throw new Error('Order not found.')
    if (order.status !== 'delivered') throw new Error('Only delivered orders are eligible for return.')
    if (lineProductIds.length === 0) throw new Error('Select at least one item to return.')
    if (!reason.trim()) throw new Error('A return reason is required.')
    return order
  }
}
