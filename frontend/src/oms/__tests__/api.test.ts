// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import * as api from '../api'
import type { APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve(body),
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('listSalesOrders', () => {
  it('builds the query string with repeated status params and fulfilled', async () => {
    const fetchMock = mockFetch({ sales_orders: [], total: 0 })

    await api.listSalesOrders(TOKEN, { fulfilled: false, status: ['pending', 'processing', 'on-hold'] })

    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain('fulfilled=false')
    expect(calledUrl).toContain('status=pending')
    expect(calledUrl).toContain('status=processing')
    expect(calledUrl).toContain('status=on-hold')
  })

  it('omits status/fulfilled entirely when not passed', async () => {
    const fetchMock = mockFetch({ sales_orders: [], total: 0 })

    await api.listSalesOrders(TOKEN, {})

    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).not.toContain('status=')
    expect(calledUrl).not.toContain('fulfilled=')
  })

  it('includes customer_id when passed, omits it otherwise', async () => {
    const fetchMock = mockFetch({ sales_orders: [], total: 0 })

    await api.listSalesOrders(TOKEN, { customerId: 'cust-1' })
    expect(fetchMock.mock.calls[0][0] as string).toContain('customer_id=cust-1')

    await api.listSalesOrders(TOKEN, {})
    expect(fetchMock.mock.calls[1][0] as string).not.toContain('customer_id')
  })
})

describe('bulkUpdateSalesOrders', () => {
  it('posts order_ids/status/fulfilled and maps the updated/failed result', async () => {
    const fetchMock = mockFetch({ updated: ['so-1'], failed: [{ order_id: 'so-2', error: 'sales order not found' }] })

    const result = await api.bulkUpdateSalesOrders(TOKEN, { orderIds: ['so-1', 'so-2'], status: 'processing' })

    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toContain('/sales-orders/bulk-status')
    expect(JSON.parse((options as RequestInit).body as string)).toEqual({ order_ids: ['so-1', 'so-2'], status: 'processing', fulfilled: undefined })
    expect(result).toEqual({ updated: ['so-1'], failed: [{ orderId: 'so-2', error: 'sales order not found' }] })
  })
})
