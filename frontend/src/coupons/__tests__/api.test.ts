import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Coupon, APIResponse } from '../types'

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

const sampleCoupon: Coupon = {
  id: 'coupon-1',
  code: 'SUMMER10',
  discount_type: 'percent',
  discount_value: 1000,
  min_subtotal_minor_units: null,
  usage_limit: null,
  times_used: 0,
  expires_at: null,
  active: true,
  created_at: '2026-06-01T00:00:00Z',
  updated_at: '2026-06-01T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listCoupons', () => {
  it('returns array of coupons', async () => {
    mockFetch([sampleCoupon])
    const result = await api.listCoupons(TOKEN)
    expect(result).toEqual([sampleCoupon])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listCoupons(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/coupons',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listCoupons(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('createCoupon', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleCoupon, 201)
    const input = { code: 'SUMMER10', discount_type: 'percent' as const, discount_value: 1000 }
    const result = await api.createCoupon(TOKEN, input)
    expect(result).toEqual(sampleCoupon)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/coupons')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'coupon code already exists', 422)
    await expect(api.createCoupon(TOKEN, {
      code: 'SUMMER10', discount_type: 'percent', discount_value: 1000,
    })).rejects.toThrow('coupon code already exists')
  })
})

describe('updateCoupon', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...sampleCoupon, discount_value: 2000 })
    const input = { code: 'SUMMER10', discount_type: 'percent' as const, discount_value: 2000, active: true }
    const result = await api.updateCoupon(TOKEN, 'coupon-1', input)
    expect(result.discount_value).toBe(2000)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/coupons/coupon-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'coupon not found', 404)
    await expect(api.updateCoupon(TOKEN, 'missing', {
      code: 'X', discount_type: 'percent', discount_value: 1000, active: true,
    })).rejects.toThrow('coupon not found')
  })
})

describe('deleteCoupon', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteCoupon(TOKEN, 'coupon-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/coupons/coupon-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'coupon not found', 404)
    await expect(api.deleteCoupon(TOKEN, 'missing')).rejects.toThrow('coupon not found')
  })
})
