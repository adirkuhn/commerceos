import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Warehouse, APIResponse } from '../types'

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

const sampleWarehouse: Warehouse = {
  id: 'wh-1',
  tenant_id: 'tenant-1',
  name: 'Main DC',
  address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
  created_at: '2026-07-26T00:00:00Z',
  updated_at: '2026-07-26T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listWarehouses', () => {
  it('returns array of warehouses', async () => {
    mockFetch([sampleWarehouse])
    const result = await api.listWarehouses(TOKEN)
    expect(result).toEqual([sampleWarehouse])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listWarehouses(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/warehouses',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listWarehouses(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('createWarehouse', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleWarehouse, 201)
    const input = { name: 'Main DC', address1: '1 Main St', city: 'Springfield', country: 'US' }
    const result = await api.createWarehouse(TOKEN, input)
    expect(result).toEqual(sampleWarehouse)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/warehouses')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'name is required', 422)
    await expect(api.createWarehouse(TOKEN, { name: '' })).rejects.toThrow('name is required')
  })
})

describe('updateWarehouse', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...sampleWarehouse, name: 'Renamed DC' })
    const input = { name: 'Renamed DC', province: 'IL', zip: '62701', phone: '+1 555-0100', active: false }
    const result = await api.updateWarehouse(TOKEN, 'wh-1', input)
    expect(result.name).toBe('Renamed DC')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/warehouses/wh-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'warehouse not found', 404)
    await expect(api.updateWarehouse(TOKEN, 'missing', { name: 'X' })).rejects.toThrow('warehouse not found')
  })
})

describe('deleteWarehouse', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteWarehouse(TOKEN, 'wh-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/warehouses/wh-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'warehouse not found', 404)
    await expect(api.deleteWarehouse(TOKEN, 'missing')).rejects.toThrow('warehouse not found')
  })
})
