import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { TenantSettings, APIResponse } from '../types'

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

const sampleSettings: TenantSettings = {
  tenant_id: 'tenant-1', default_currency: 'USD', timezone: 'UTC', date_format: 'YYYY-MM-DD', language: 'en',
  updated_at: '2026-07-26T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('getTenantSettings', () => {
  it('returns the tenant settings', async () => {
    mockFetch(sampleSettings)
    const result = await api.getTenantSettings(TOKEN)
    expect(result).toEqual(sampleSettings)
  })

  it('calls the correct endpoint', async () => {
    mockFetch(sampleSettings)
    await api.getTenantSettings(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/settings/tenant',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.getTenantSettings(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('updateTenantSettings', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...sampleSettings, default_currency: 'EUR' })
    const input = { default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl' }
    const result = await api.updateTenantSettings(TOKEN, input)
    expect(result.default_currency).toBe('EUR')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/settings/tenant')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.updateTenantSettings(TOKEN, { default_currency: 'EUR' })).rejects.toThrow('internal server error')
  })
})
