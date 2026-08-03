import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Connection, SyncSchedule, APIResponse } from '../types'

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

const sampleConnection: Connection = {
  id: 'conn-1',
  provider: 'woocommerce',
  store_url: 'https://mystore.com',
  status: 'connected',
  created_at: '2026-07-26T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listCatalog', () => {
  it('returns catalog entries', async () => {
    mockFetch([{ provider: 'woocommerce', name: 'WooCommerce' }])
    const result = await api.listCatalog(TOKEN)
    expect(result).toEqual([{ provider: 'woocommerce', name: 'WooCommerce' }])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listCatalog(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/catalog',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('listConnections', () => {
  it('returns array of connections', async () => {
    mockFetch([sampleConnection])
    const result = await api.listConnections(TOKEN)
    expect(result).toHaveLength(1)
    expect(result[0].store_url).toBe('https://mystore.com')
  })

  it('never needs to handle a secret field (backend never sends one)', async () => {
    mockFetch([sampleConnection])
    const result = await api.listConnections(TOKEN)
    expect(result[0]).not.toHaveProperty('consumer_secret')
    expect(result[0]).not.toHaveProperty('consumer_secret_encrypted')
  })
})

describe('startWooConnect', () => {
  it('returns authorize_url on success', async () => {
    mockFetch({ authorize_url: 'https://mystore.com/wc-auth/v1/authorize?...' })
    const result = await api.startWooConnect(TOKEN, 'https://mystore.com')
    expect(result.authorize_url).toContain('wc-auth/v1/authorize')
  })

  it('sends store_url in body', async () => {
    mockFetch({ authorize_url: 'https://mystore.com/wc-auth/v1/authorize?...' })
    await api.startWooConnect(TOKEN, 'https://mystore.com')
    const fetchMock = vi.mocked(fetch)
    const [, opts] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({ store_url: 'https://mystore.com' })
  })

  it('throws on invalid store url', async () => {
    mockFetchError('VALIDATION_ERROR', 'store URL must be a valid http(s) URL', 422)
    await expect(api.startWooConnect(TOKEN, 'not-a-url')).rejects.toThrow('store URL must be a valid http(s) URL')
  })
})

describe('disconnectConnection', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.disconnectConnection(TOKEN, 'conn-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'integration connection not found', 404)
    await expect(api.disconnectConnection(TOKEN, 'missing')).rejects.toThrow('integration connection not found')
  })
})

describe('testConnection', () => {
  it('returns updated connection on success', async () => {
    mockFetch(sampleConnection)
    const result = await api.testConnection(TOKEN, 'conn-1')
    expect(result.status).toBe('connected')
  })

  it('calls correct endpoint', async () => {
    mockFetch(sampleConnection)
    await api.testConnection(TOKEN, 'conn-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/test',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('throws on upstream WooCommerce failure', async () => {
    mockFetchError('UPSTREAM_ERROR', 'connection test failed: 401 unauthorized', 502)
    await expect(api.testConnection(TOKEN, 'conn-1')).rejects.toThrow('connection test failed')
  })
})

describe('manualConnectWooCommerce', () => {
  it('returns the new connection on success', async () => {
    mockFetch(sampleConnection)
    const result = await api.manualConnectWooCommerce(TOKEN, {
      storeUrl: 'https://mystore.com',
      consumerKey: 'ck_abc',
      consumerSecret: 'cs_abc',
    })
    expect(result.status).toBe('connected')
  })

  it('sends store_url/consumer_key/consumer_secret in body', async () => {
    mockFetch(sampleConnection)
    await api.manualConnectWooCommerce(TOKEN, {
      storeUrl: 'https://mystore.com',
      consumerKey: 'ck_abc',
      consumerSecret: 'cs_abc',
    })
    const fetchMock = vi.mocked(fetch)
    const [url, opts] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/woocommerce/connect/manual')
    expect(JSON.parse(opts.body as string)).toEqual({
      store_url: 'https://mystore.com',
      consumer_key: 'ck_abc',
      consumer_secret: 'cs_abc',
    })
  })

  it('throws on invalid store url', async () => {
    mockFetchError('VALIDATION_ERROR', 'store URL must be a valid http(s) URL', 422)
    await expect(api.manualConnectWooCommerce(TOKEN, {
      storeUrl: 'not-a-url',
      consumerKey: 'ck_abc',
      consumerSecret: 'cs_abc',
    })).rejects.toThrow('store URL must be a valid http(s) URL')
  })
})

const sampleSchedule: SyncSchedule = {
  entity_type: 'products', direction: 'import', enabled: true, interval_minutes: 15,
  default_warehouse_id: 'wh-1', default_currency: 'USD',
  last_run_at: null, last_run_status: '', last_run_error: '',
}

describe('listSyncSchedules', () => {
  it('returns schedules for a connection', async () => {
    mockFetch([sampleSchedule])
    const result = await api.listSyncSchedules(TOKEN, 'conn-1')
    expect(result).toEqual([sampleSchedule])
  })

  it('calls the correct endpoint', async () => {
    mockFetch([])
    await api.listSyncSchedules(TOKEN, 'conn-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/sync-schedules',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'integration connection not found', 404)
    await expect(api.listSyncSchedules(TOKEN, 'missing')).rejects.toThrow('integration connection not found')
  })
})

describe('upsertSyncSchedules', () => {
  it('sends schedules in body to the correct endpoint', async () => {
    mockFetch([sampleSchedule])
    await api.upsertSyncSchedules(TOKEN, 'conn-1', [sampleSchedule])
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/sync-schedules')
    expect(opts.method).toBe('PUT')
    expect(JSON.parse(opts.body as string)).toEqual({ schedules: [sampleSchedule] })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'this entity type does not support scheduled sync yet', 422)
    await expect(api.upsertSyncSchedules(TOKEN, 'conn-1', [sampleSchedule])).rejects.toThrow('this entity type does not support scheduled sync yet')
  })
})

describe('importWooProducts', () => {
  it('sends default_warehouse_id/default_currency in body to the correct endpoint', async () => {
    mockFetch({ imported: 3, already_linked: 1, skipped_type: 0, errors: [] })
    const result = await api.importWooProducts(TOKEN, 'conn-1', 'wh-1', 'USD')
    expect(result.imported).toBe(3)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/import')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual({ default_warehouse_id: 'wh-1', default_currency: 'USD' })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'default_warehouse_id and default_currency are required', 422)
    await expect(api.importWooProducts(TOKEN, 'conn-1', '', '')).rejects.toThrow('default_warehouse_id and default_currency are required')
  })
})

describe('deleteAllImportedProducts', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch({ deleted_products: 805, deleted_categories: 12, errors: [] })
    const result = await api.deleteAllImportedProducts(TOKEN, 'conn-1')
    expect(result.deleted_products).toBe(805)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/imported-products',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on failure', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.deleteAllImportedProducts(TOKEN, 'conn-1')).rejects.toThrow('internal server error')
  })
})

describe('updateConnectionSettings', () => {
  it('sends default_currency in body to the correct endpoint', async () => {
    mockFetch({ ...sampleConnection, default_currency: 'USD' })
    const result = await api.updateConnectionSettings(TOKEN, 'conn-1', 'USD')
    expect(result.default_currency).toBe('USD')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/settings')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual({ default_currency: 'USD' })
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'integration connection not found', 404)
    await expect(api.updateConnectionSettings(TOKEN, 'missing', 'USD')).rejects.toThrow('integration connection not found')
  })
})
