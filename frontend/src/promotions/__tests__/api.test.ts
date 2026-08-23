import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Promotion, PromotionDetail, APIResponse } from '../types'

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

const samplePromotion: Promotion = {
  id: 'promo-1',
  name: 'Summer Acai',
  type: 'percentage',
  starts_at: '2026-07-01T00:00:00Z',
  ends_at: '2026-07-31T00:00:00Z',
  status: 'active',
  created_at: '2026-06-01T00:00:00Z',
  updated_at: '2026-06-01T00:00:00Z',
}

const samplePromotionDetail: PromotionDetail = {
  ...samplePromotion,
  lines: [{ id: 'line-1', sku_id: 'sku-1', sku_code: 'ACAI-1L', discount_type: 'percentage', discount_value: 1000 }],
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listPromotions', () => {
  it('returns array of promotions', async () => {
    mockFetch([samplePromotion])
    const result = await api.listPromotions(TOKEN)
    expect(result).toEqual([samplePromotion])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listPromotions(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/promotions',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listPromotions(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('getPromotion', () => {
  it('returns promotion detail with lines', async () => {
    mockFetch(samplePromotionDetail)
    const result = await api.getPromotion(TOKEN, 'promo-1')
    expect(result.lines).toHaveLength(1)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/promotions/promo-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('createPromotion', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(samplePromotionDetail, 200)
    const input = {
      name: 'Summer Acai', type: 'percentage' as const, starts_at: '2026-07-01T00:00:00Z',
      lines: [{ sku_id: 'sku-1', discount_type: 'percentage' as const, discount_value: 1000 }],
    }
    const result = await api.createPromotion(TOKEN, input)
    expect(result).toEqual(samplePromotionDetail)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/promotions')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'promotion must include at least one sku', 422)
    await expect(api.createPromotion(TOKEN, {
      name: 'Promo', type: 'percentage', starts_at: '2026-07-01T00:00:00Z', lines: [],
    })).rejects.toThrow('promotion must include at least one sku')
  })
})

describe('updatePromotion', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...samplePromotionDetail, name: 'Summer Acai v2' })
    const input = {
      name: 'Summer Acai v2', type: 'percentage' as const, starts_at: '2026-07-01T00:00:00Z',
      status: 'active' as const, lines: [{ sku_id: 'sku-1', discount_type: 'percentage' as const, discount_value: 1000 }],
    }
    const result = await api.updatePromotion(TOKEN, 'promo-1', input)
    expect(result.name).toBe('Summer Acai v2')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/promotions/promo-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'promotion not found', 404)
    await expect(api.updatePromotion(TOKEN, 'missing', {
      name: 'X', type: 'percentage', starts_at: '2026-07-01T00:00:00Z', status: 'active', lines: [],
    })).rejects.toThrow('promotion not found')
  })
})

describe('deletePromotion', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deletePromotion(TOKEN, 'promo-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/promotions/promo-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'promotion not found', 404)
    await expect(api.deletePromotion(TOKEN, 'missing')).rejects.toThrow('promotion not found')
  })
})
