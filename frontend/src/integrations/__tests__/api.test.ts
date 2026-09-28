import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Connection, SyncSchedule, ProductChannelListing, SyncLogEntry, APIResponse, SalesChannel, SalesChannelProductEnrollment } from '../types'

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
  it('sends default_currency and default_warehouse_id:null in body to the correct endpoint', async () => {
    mockFetch({ ...sampleConnection, default_currency: 'USD' })
    const result = await api.updateConnectionSettings(TOKEN, 'conn-1', 'USD')
    expect(result.default_currency).toBe('USD')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/settings')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual({
      default_currency: 'USD', default_warehouse_id: null,
      bol_economic_operator_id: null, bol_delivery_code: null,
    })
  })

  it('sends the given default_warehouse_id', async () => {
    mockFetch({ ...sampleConnection, default_currency: 'USD', default_warehouse_id: 'wh-1' })
    await api.updateConnectionSettings(TOKEN, 'conn-1', 'USD', 'wh-1')
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({
      default_currency: 'USD', default_warehouse_id: 'wh-1',
      bol_economic_operator_id: null, bol_delivery_code: null,
    })
  })

  it('sends the given bol_economic_operator_id and bol_delivery_code', async () => {
    mockFetch(sampleConnection)
    await api.updateConnectionSettings(TOKEN, 'conn-1', '', null, 'op-1', '1-2d')
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({
      default_currency: '', default_warehouse_id: null,
      bol_economic_operator_id: 'op-1', bol_delivery_code: '1-2d',
    })
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'integration connection not found', 404)
    await expect(api.updateConnectionSettings(TOKEN, 'missing', 'USD')).rejects.toThrow('integration connection not found')
  })
})

const sampleListing: ProductChannelListing = {
  id: 'listing-1',
  product_id: 'prod-1',
  connection_id: 'conn-1',
  provider: 'woocommerce',
  external_id: '23609',
  sync_status: 'synced',
  last_synced_at: '2026-07-26T00:00:00Z',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-26T00:00:00Z',
}

describe('getProductChannelListing', () => {
  it('returns the listing when the product is linked', async () => {
    mockFetch(sampleListing)
    const result = await api.getProductChannelListing(TOKEN, 'conn-1', 'prod-1')
    expect(result).toEqual(sampleListing)
  })

  it('calls the correct endpoint', async () => {
    mockFetch(sampleListing)
    await api.getProductChannelListing(TOKEN, 'conn-1', 'prod-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/listings/prod-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('returns null (not a thrown error) when the product was never linked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve({ data: null, error: { code: 'NOT_FOUND', message: 'listing not found' } }),
    }))
    const result = await api.getProductChannelListing(TOKEN, 'conn-1', 'prod-1')
    expect(result).toBeNull()
  })

  it('throws on unexpected failure', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.getProductChannelListing(TOKEN, 'conn-1', 'prod-1')).rejects.toThrow('internal server error')
  })
})

const sampleLogEntry: SyncLogEntry = {
  id: 'log-1', listing_id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1',
  provider: 'woocommerce', external_id: '23609', direction: 'push', status: 'success',
  error_message: '', fields: { name: 'T-Shirt' }, changes: { name: { from: '', to: 'T-Shirt' } },
  synced_at: '2026-08-04T10:00:00Z',
}

function mockSyncLogListFetch(data: SyncLogEntry[], total: number) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ data, total, error: null }),
  }))
}

describe('listSyncLogs', () => {
  it('returns entries for a listing', async () => {
    mockFetch([sampleLogEntry])
    const result = await api.listSyncLogs(TOKEN, 'conn-1', 'prod-1')
    expect(result).toEqual([sampleLogEntry])
  })

  it('calls the correct endpoint', async () => {
    mockFetch([])
    await api.listSyncLogs(TOKEN, 'conn-1', 'prod-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/listings/prod-1/sync-logs',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('listAllSyncLogs', () => {
  it('returns entries and total', async () => {
    mockSyncLogListFetch([sampleLogEntry], 42)
    const result = await api.listAllSyncLogs(TOKEN, { page: 2, perPage: 10 })
    expect(result).toEqual({ entries: [sampleLogEntry], total: 42 })
  })

  it('sends page/per_page as query params', async () => {
    mockSyncLogListFetch([], 0)
    await api.listAllSyncLogs(TOKEN, { page: 2, perPage: 10 })
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/sync-logs?page=2&per_page=10')
  })

  it('omits query params when not given', async () => {
    mockSyncLogListFetch([], 0)
    await api.listAllSyncLogs(TOKEN)
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/sync-logs')
  })

  it('throws on failure', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listAllSyncLogs(TOKEN)).rejects.toThrow('internal server error')
  })
})

const sampleChannel: SalesChannel = {
  id: 'chan-1',
  name: 'My Woo Store',
  type: 'woocommerce',
  connection_id: 'conn-1',
  enrollment_mode: 'all',
  active: true,
  created_at: '2026-08-01T00:00:00Z',
  updated_at: '2026-08-01T00:00:00Z',
}

describe('listSalesChannels', () => {
  it('returns array of channels', async () => {
    mockFetch([sampleChannel])
    const result = await api.listSalesChannels(TOKEN)
    expect(result).toEqual([sampleChannel])
  })

  it('calls the correct endpoint', async () => {
    mockFetch([])
    await api.listSalesChannels(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/sales-channels',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('createSalesChannel', () => {
  it('sends the payload to the correct endpoint', async () => {
    mockFetch(sampleChannel)
    await api.createSalesChannel(TOKEN, {
      name: 'My Woo Store', type: 'woocommerce', connectionId: 'conn-1', enrollmentMode: 'all',
    })
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/sales-channels')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual({
      name: 'My Woo Store', type: 'woocommerce', connection_id: 'conn-1', enrollment_mode: 'all', quantity_breaks: [],
    })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'connection is required for this channel type', 422)
    await expect(api.createSalesChannel(TOKEN, { name: 'x', type: 'woocommerce', enrollmentMode: 'all' })).rejects.toThrow(
      'connection is required for this channel type',
    )
  })

  it('sends quantity breaks through', async () => {
    mockFetch(sampleChannel)
    await api.createSalesChannel(TOKEN, {
      name: 'B2B Store', type: 'internal', enrollmentMode: 'all',
      quantityBreaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    })
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string).quantity_breaks).toEqual([{ min_quantity: 10, discount_basis_points: 1500 }])
  })

  it('sends the storefront mode through', async () => {
    mockFetch(sampleChannel)
    await api.createSalesChannel(TOKEN, { name: 'Shop', type: 'internal', enrollmentMode: 'all', storefrontMode: 'theme' })
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string).storefront_mode).toBe('theme')
  })
})

describe('updateSalesChannel', () => {
  it('sends the payload to the correct endpoint', async () => {
    mockFetch({ ...sampleChannel, name: 'Renamed' })
    await api.updateSalesChannel(TOKEN, 'chan-1', {
      name: 'Renamed', connectionId: 'conn-1', enrollmentMode: 'tag', enrollmentTag: 'summer', active: true,
    })
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/sales-channels/chan-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual({
      name: 'Renamed', connection_id: 'conn-1', enrollment_mode: 'tag', enrollment_tag: 'summer', active: true, quantity_breaks: [],
    })
  })

  it('sends the storefront mode through', async () => {
    mockFetch(sampleChannel)
    await api.updateSalesChannel(TOKEN, 'chan-1', { name: 'Shop', enrollmentMode: 'all', active: true, storefrontMode: 'headless', signInUrl: '' })
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string).storefront_mode).toBe('headless')
    // "" clears the sign-in page; omitting it keeps the current one.
    expect(JSON.parse(opts.body as string).sign_in_url).toBe('')
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'sales channel not found', 404)
    await expect(api.updateSalesChannel(TOKEN, 'missing', { name: 'x', enrollmentMode: 'all', active: true })).rejects.toThrow(
      'sales channel not found',
    )
  })
})

describe('deleteSalesChannel', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteSalesChannel(TOKEN, 'chan-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/integrations/sales-channels/chan-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })
})

const sampleEnrollment: SalesChannelProductEnrollment = {
  product: { id: 'p1', name: 'T-Shirt', slug: 't-shirt', category_id: 'cat-1', tags: ['summer'] },
  enrolled: true,
}

function mockChannelProductsFetch(data: SalesChannelProductEnrollment[], total: number) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ data, total, error: null }),
  }))
}

describe('listSalesChannelProducts', () => {
  it('returns items and total', async () => {
    mockChannelProductsFetch([sampleEnrollment], 1)
    const result = await api.listSalesChannelProducts(TOKEN, 'chan-1')
    expect(result).toEqual({ items: [sampleEnrollment], total: 1 })
  })

  it('sends search/page/per_page as query params', async () => {
    mockChannelProductsFetch([], 0)
    await api.listSalesChannelProducts(TOKEN, 'chan-1', { search: 'shirt', page: 2, perPage: 20 })
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/sales-channels/chan-1/products?search=shirt&page=2&per_page=20')
  })

  it('throws on failure', async () => {
    mockFetchError('NOT_FOUND', 'sales channel not found', 404)
    await expect(api.listSalesChannelProducts(TOKEN, 'missing')).rejects.toThrow('sales channel not found')
  })
})

describe('setSalesChannelProductOverride', () => {
  it('sends included in body to the correct endpoint', async () => {
    mockFetch(null)
    await api.setSalesChannelProductOverride(TOKEN, 'chan-1', 'p1', true)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/sales-channels/chan-1/products/p1/override')
    expect(opts.method).toBe('PUT')
    expect(JSON.parse(opts.body as string)).toEqual({ included: true })
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'product not found', 404)
    await expect(api.setSalesChannelProductOverride(TOKEN, 'chan-1', 'missing', true)).rejects.toThrow('product not found')
  })
})
