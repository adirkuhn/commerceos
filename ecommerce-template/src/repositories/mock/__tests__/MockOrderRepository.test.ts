import { describe, expect, it } from 'vitest'
import { MockOrderRepository } from '../MockOrderRepository'
import { DEMO_CUSTOMER } from '@/config/demoData'

const repo = new MockOrderRepository()

describe('MockOrderRepository', () => {
  it('lists orders for a customer, newest first', async () => {
    const orders = await repo.listForCustomer(DEMO_CUSTOMER.id)
    expect(orders.length).toBeGreaterThan(0)
    for (let i = 1; i < orders.length; i++) {
      expect(new Date(orders[i - 1].placedAt).getTime()).toBeGreaterThanOrEqual(new Date(orders[i].placedAt).getTime())
    }
  })

  it('returns an empty list for a customer with no orders', async () => {
    expect(await repo.listForCustomer('nobody')).toEqual([])
  })

  it('finds an order by its own id', async () => {
    const [first] = await repo.listForCustomer(DEMO_CUSTOMER.id)
    const found = await repo.getById(first.id)
    expect(found?.id).toBe(first.id)
  })

  it('looks up a guest/demo order by order number + matching email', async () => {
    const [first] = await repo.listForCustomer(DEMO_CUSTOMER.id)
    const found = await repo.findByOrderNumberAndEmail(first.orderNumber, DEMO_CUSTOMER.email)
    expect(found?.id).toBe(first.id)
  })

  it('rejects a lookup with the right order number but the wrong email', async () => {
    const [first] = await repo.listForCustomer(DEMO_CUSTOMER.id)
    expect(await repo.findByOrderNumberAndEmail(first.orderNumber, 'someone-else@example.com')).toBeNull()
  })

  it('cancels a pending/processing order, rejects cancelling a delivered one', async () => {
    const orders = await repo.listForCustomer(DEMO_CUSTOMER.id)
    const processing = orders.find(o => o.status === 'processing')!
    const delivered = orders.find(o => o.status === 'delivered')!

    const cancelled = await repo.cancel(processing.id)
    expect(cancelled.status).toBe('cancelled')
    await expect(repo.cancel(delivered.id)).rejects.toThrow()
  })

  it('requests a return only for a delivered order with items and a reason', async () => {
    const orders = await repo.listForCustomer(DEMO_CUSTOMER.id)
    const delivered = orders.find(o => o.status === 'delivered')!
    const notDelivered = orders.find(o => o.status !== 'delivered')!

    await expect(repo.requestReturn(delivered.id, [], 'Wrong size')).rejects.toThrow()
    await expect(repo.requestReturn(delivered.id, [delivered.lines[0].productId], '')).rejects.toThrow()
    await expect(repo.requestReturn(notDelivered.id, [notDelivered.lines[0].productId], 'Wrong size')).rejects.toThrow()
    const ok = await repo.requestReturn(delivered.id, [delivered.lines[0].productId], 'Wrong size')
    expect(ok.id).toBe(delivered.id)
  })
})
