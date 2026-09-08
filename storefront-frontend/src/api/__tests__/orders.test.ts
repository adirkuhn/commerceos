import { describe, it, expect, vi, beforeEach } from 'vitest'
import { lookupOrder } from '../orders'
import type { APIResponse, StorefrontOrderStatus } from '../../types'

function mockFetch<T>(data: T, ok = true) {
  const body: APIResponse<T> = { data: ok ? data : null, error: ok ? null : { code: 'NOT_FOUND', message: 'order not found' } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok, status: ok ? 200 : 404, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('lookupOrder', () => {
  it('sends order_number and email as query params', async () => {
    const status: StorefrontOrderStatus = { order_number: 'WEB-1', status: 'processing', fulfilled: false, total_minor_units: 1000, currency_code: 'EUR' }
    mockFetch(status)
    expect(await lookupOrder('WEB-1', 'shopper@example.com')).toEqual(status)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('order_number=WEB-1')
    expect(url).toContain('email=shopper%40example.com')
  })

  it('throws not-found on a bad number/email combination', async () => {
    mockFetch(null, false)
    await expect(lookupOrder('WEB-2', 'nope@example.com')).rejects.toThrow('order not found')
  })
})
