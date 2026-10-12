// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { APIResponse, DashboardStatsReport } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 500) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const sampleReport: DashboardStatsReport = {
  total_stock_units: 84293,
  currency_code: 'EUR',
  revenue_this_month_minor_units: 11600000,
  stock_flow_last_7_days: [{ date: '2026-08-16', in: 10, out: 5 }],
  sales_by_channel_this_month: [{ source: 'woocommerce', total_minor_units: 3150000 }],
  top_products_this_month: [{ sku_id: 'sku-1', sku_code: 'WHP-001', product_name: 'Headphones', quantity_sold: 234, revenue_minor_units: 5850000 }],
}

beforeEach(() => { vi.restoreAllMocks() })

describe('getDashboardStats', () => {
  it('returns the report', async () => {
    mockFetch(sampleReport)
    const result = await api.getDashboardStats(TOKEN)
    expect(result).toEqual(sampleReport)
  })

  it('sends the bearer token', async () => {
    mockFetch(sampleReport)
    await api.getDashboardStats(TOKEN)
    const [, options] = vi.mocked(fetch).mock.calls[0]
    expect((options?.headers as Record<string, string>).Authorization).toBe(`Bearer ${TOKEN}`)
  })

  it('throws on an API error', async () => {
    mockFetchError('INTERNAL', 'internal server error')
    await expect(api.getDashboardStats(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('getTodayQueue', () => {
  const queue = { to_pick: 4, to_pack: 2, stock_issues: 1, unmatched_skus: 0, expiring_batches: 3, expired_batches: 0 }

  it('GETs /dashboard/today with the bearer token and returns the counts', async () => {
    mockFetch(queue)
    const result = await api.getTodayQueue(TOKEN)
    expect(result).toEqual(queue)
    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(String(url)).toMatch(/\/dashboard\/today$/)
    expect((init as RequestInit).headers).toMatchObject({ Authorization: `Bearer ${TOKEN}` })
  })

  it('throws the coded API error on failure', async () => {
    mockFetchError('INTERNAL', 'internal server error')
    await expect(api.getTodayQueue(TOKEN)).rejects.toThrow()
  })
})
