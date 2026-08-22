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
})
