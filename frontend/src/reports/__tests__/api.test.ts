import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { SalesSummaryReport, ProductPerformanceReport, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('getSalesSummaryReport', () => {
  const sample: SalesSummaryReport = {
    currency_code: 'EUR',
    current: { revenue_minor_units: 118000, order_count: 112, units_sold: 217, average_order_value_minor_units: 1053 },
    previous: { revenue_minor_units: 100000, order_count: 100, units_sold: 178, average_order_value_minor_units: 1000 },
    revenue_change_percent: 18.4, order_count_change_percent: 12.1, units_sold_change_percent: 21.9, aov_change_percent: 5.3,
  }

  it('returns the report', async () => {
    mockFetch(sample)
    const result = await api.getSalesSummaryReport(TOKEN, '2026-08-01T00:00:00Z', '2026-08-31T00:00:00Z')
    expect(result).toEqual(sample)
  })

  it('sends from/to as query params to the correct endpoint', async () => {
    mockFetch(sample)
    await api.getSalesSummaryReport(TOKEN, '2026-08-01T00:00:00Z', '2026-08-31T00:00:00Z')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/reports/sales-summary?from=2026-08-01T00%3A00%3A00Z&to=2026-08-31T00%3A00%3A00Z',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'report end date must be after the start date', 422)
    await expect(api.getSalesSummaryReport(TOKEN, '2026-08-31T00:00:00Z', '2026-08-01T00:00:00Z')).rejects.toThrow(
      'report end date must be after the start date',
    )
  })
})

describe('getProductPerformanceReport', () => {
  const sample: ProductPerformanceReport = {
    currency_code: 'EUR',
    rows: [{
      sku_id: 'sku-1', sku_code: 'ACAI-1L', product_name: 'Acai 1L', quantity_sold: 5, revenue_minor_units: 2500,
      order_count: 2, first_sale_at: '2026-08-01T00:00:00Z', last_sale_at: '2026-08-15T00:00:00Z',
      cost_price_minor_units: 300, profit_minor_units: 1000, margin_basis_points: 4000,
    }],
  }

  it('returns the report', async () => {
    mockFetch(sample)
    const result = await api.getProductPerformanceReport(TOKEN, '2026-08-01T00:00:00Z', '2026-08-31T00:00:00Z')
    expect(result).toEqual(sample)
  })

  it('calls the correct endpoint', async () => {
    mockFetch(sample)
    await api.getProductPerformanceReport(TOKEN, '2026-08-01T00:00:00Z', '2026-08-31T00:00:00Z')
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe(
      'https://api.commerceos.dev.localhost/api/v1/reports/product-performance?from=2026-08-01T00%3A00%3A00Z&to=2026-08-31T00%3A00%3A00Z',
    )
  })
})
