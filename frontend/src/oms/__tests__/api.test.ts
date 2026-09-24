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

describe('listSalesOrders awaitingPicking', () => {
  it('sends awaiting_picking=true when requested, omits it otherwise', async () => {
    const fetchMock = mockFetch({ sales_orders: [], total: 0 })

    await api.listSalesOrders(TOKEN, { awaitingPicking: true })
    expect(fetchMock.mock.calls[0][0] as string).toContain('awaiting_picking=true')

    await api.listSalesOrders(TOKEN, {})
    expect(fetchMock.mock.calls[1][0] as string).not.toContain('awaiting_picking')
  })
})

describe('listPackingQueue', () => {
  it('requests the packing queue page and returns entries with locks', async () => {
    const queue = { entries: [{ sales_order: { id: 'so-1' }, lock: { activity: 'picking', user_id: 'u1', since: '2026-09-24T10:00:00Z' } }], total: 1 }
    const fetchMock = mockFetch(queue)

    const result = await api.listPackingQueue(TOKEN, { perPage: 100 })

    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toMatch(/\/packing-queue\?per_page=100$/)
    expect(result).toEqual(queue)
  })
})

describe('markOrderPicked', () => {
  it('posts to the order mark-picked endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, json: () => Promise.resolve(null) })
    vi.stubGlobal('fetch', fetchMock)

    await api.markOrderPicked(TOKEN, 'so-1')

    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/sales-orders\/so-1\/mark-picked$/)
    expect((options as RequestInit).method).toBe('POST')
  })
})

describe('getPickList', () => {
  it('repeats sales_order_id per order and returns the pick list', async () => {
    const pickList = { items: [{ sku_id: 'sku-a', sku_code: 'A', barcode: '', name: 'Apple', total_quantity: 3, orders: [] }] }
    const fetchMock = mockFetch(pickList)

    const result = await api.getPickList(TOKEN, ['so-1', 'so-2'])

    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain('/sales-orders/pick-list?')
    expect(url).toContain('sales_order_id=so-1&sales_order_id=so-2')
    expect(result).toEqual(pickList)
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
