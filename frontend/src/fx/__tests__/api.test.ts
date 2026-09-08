import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { ExchangeRate, APIResponse } from '../types'

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

const sampleRate: ExchangeRate = { currency_code: 'USD', rate_basis_points: 10876, updated_at: '2026-09-08T00:00:00Z' }

beforeEach(() => { vi.restoreAllMocks() })

describe('listExchangeRates', () => {
  it('returns the rates', async () => {
    mockFetch([sampleRate])
    const result = await api.listExchangeRates(TOKEN)
    expect(result).toEqual([sampleRate])
  })

  it('calls the correct endpoint', async () => {
    mockFetch([sampleRate])
    await api.listExchangeRates(TOKEN)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/fx/rates')
    expect(options?.method).toBe('GET')
  })
})

describe('upsertExchangeRate', () => {
  it('PUTs the currency code and rate', async () => {
    mockFetch(sampleRate)
    const result = await api.upsertExchangeRate(TOKEN, { currency_code: 'USD', rate_basis_points: 10876 })
    expect(result).toEqual(sampleRate)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/fx/rates')
    expect(options?.method).toBe('PUT')
    expect(JSON.parse(options?.body as string)).toEqual({ currency_code: 'USD', rate_basis_points: 10876 })
  })

  it('throws with the server error message on failure', async () => {
    mockFetchError('VALIDATION_ERROR', 'currency code must be 3 uppercase letters')
    await expect(api.upsertExchangeRate(TOKEN, { currency_code: 'us', rate_basis_points: 10876 })).rejects.toThrow('currency code must be 3 uppercase letters')
  })
})

describe('deleteExchangeRate', () => {
  it('DELETEs by currency code', async () => {
    mockFetch(null)
    await api.deleteExchangeRate(TOKEN, 'USD')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/fx/rates/USD')
    expect(options?.method).toBe('DELETE')
  })
})
