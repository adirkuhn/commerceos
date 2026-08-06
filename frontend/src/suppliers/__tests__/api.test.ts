import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Supplier, APIResponse } from '../types'

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

const sampleSupplier: Supplier = {
  id: 'supplier-1',
  tenant_id: 'tenant-1',
  name: 'Acme Foods',
  email: 'hi@acme.example',
  phone: '',
  country: 'US',
  lead_time_days: 14,
  status: 'active',
  notes: '',
  created_at: '2026-08-05T00:00:00Z',
  updated_at: '2026-08-05T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listSuppliers', () => {
  it('returns array of suppliers', async () => {
    mockFetch([sampleSupplier])
    const result = await api.listSuppliers(TOKEN)
    expect(result).toEqual([sampleSupplier])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listSuppliers(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/suppliers',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listSuppliers(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('createSupplier', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleSupplier, 201)
    const input = { name: 'Acme Foods', email: 'hi@acme.example', country: 'US', lead_time_days: 14, status: 'active' as const }
    const result = await api.createSupplier(TOKEN, input)
    expect(result).toEqual(sampleSupplier)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/suppliers')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'name is required', 422)
    await expect(api.createSupplier(TOKEN, { name: '', status: 'active' })).rejects.toThrow('name is required')
  })
})

describe('updateSupplier', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...sampleSupplier, name: 'Acme Foods Ltd' })
    const input = { name: 'Acme Foods Ltd', status: 'inactive' as const }
    const result = await api.updateSupplier(TOKEN, 'supplier-1', input)
    expect(result.name).toBe('Acme Foods Ltd')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/suppliers/supplier-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'supplier not found', 404)
    await expect(api.updateSupplier(TOKEN, 'missing', { name: 'X', status: 'active' })).rejects.toThrow('supplier not found')
  })
})

describe('deleteSupplier', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteSupplier(TOKEN, 'supplier-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/suppliers/supplier-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'supplier not found', 404)
    await expect(api.deleteSupplier(TOKEN, 'missing')).rejects.toThrow('supplier not found')
  })
})
