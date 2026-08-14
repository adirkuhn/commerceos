import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { TaxClass, TaxRate, APIResponse } from '../types'

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

const sampleClass: TaxClass = {
  id: 'class-1', name: 'Standard', is_default: true, created_at: '2026-07-26T00:00:00Z', updated_at: '2026-07-26T00:00:00Z',
}

const sampleRate: TaxRate = {
  id: 'rate-1', tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2100,
  created_at: '2026-07-26T00:00:00Z', updated_at: '2026-07-26T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listTaxClasses', () => {
  it('returns the tax classes', async () => {
    mockFetch([sampleClass])
    const result = await api.listTaxClasses(TOKEN)
    expect(result).toEqual([sampleClass])
  })

  it('calls the correct endpoint', async () => {
    mockFetch([sampleClass])
    await api.listTaxClasses(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/tax/classes',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listTaxClasses(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('createTaxClass', () => {
  it('sends the input to the correct endpoint', async () => {
    mockFetch(sampleClass)
    const result = await api.createTaxClass(TOKEN, { name: 'Standard', is_default: true })
    expect(result).toEqual(sampleClass)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/tax/classes')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual({ name: 'Standard', is_default: true })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'tax class name is required', 422)
    await expect(api.createTaxClass(TOKEN, { name: '' })).rejects.toThrow('tax class name is required')
  })
})

describe('updateTaxClass', () => {
  it('sends a PATCH to the class-specific endpoint', async () => {
    mockFetch({ ...sampleClass, name: 'Standard rate' })
    const result = await api.updateTaxClass(TOKEN, 'class-1', { name: 'Standard rate', is_default: true })
    expect(result.name).toBe('Standard rate')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/tax/classes/class-1')
    expect(opts.method).toBe('PATCH')
  })
})

describe('deleteTaxClass', () => {
  it('sends a DELETE to the class-specific endpoint', async () => {
    mockFetch(null)
    await api.deleteTaxClass(TOKEN, 'class-1')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/tax/classes/class-1')
    expect(opts.method).toBe('DELETE')
  })

  it('throws when the class is in use', async () => {
    mockFetchError('VALIDATION_ERROR', 'tax class is assigned to one or more SKUs', 422)
    await expect(api.deleteTaxClass(TOKEN, 'class-1')).rejects.toThrow('tax class is assigned to one or more SKUs')
  })
})

describe('listTaxRates', () => {
  it('returns the tax rates', async () => {
    mockFetch([sampleRate])
    const result = await api.listTaxRates(TOKEN)
    expect(result).toEqual([sampleRate])
  })
})

describe('upsertTaxRate', () => {
  it('sends a PUT to the rates endpoint', async () => {
    mockFetch(sampleRate)
    const result = await api.upsertTaxRate(TOKEN, { tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2100 })
    expect(result).toEqual(sampleRate)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/tax/rates')
    expect(opts.method).toBe('PUT')
    expect(JSON.parse(opts.body as string)).toEqual({ tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2100 })
  })
})

describe('deleteTaxRate', () => {
  it('sends a DELETE to the rate-specific endpoint', async () => {
    mockFetch(null)
    await api.deleteTaxRate(TOKEN, 'rate-1')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/tax/rates/rate-1')
    expect(opts.method).toBe('DELETE')
  })
})
