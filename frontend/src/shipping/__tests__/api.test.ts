import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { APIResponse, Shipment } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 502) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

beforeEach(() => { vi.restoreAllMocks() })

const sampleShipment: Shipment = {
  id: 's-1', sales_order_id: 'order-1', connection_id: 'conn-1', carrier: 1,
  barcode: '3SMYPA1', status: 1, status_main: 'registered', label_format: 'pdf',
  recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
  created_at: '2026-08-23T00:00:00Z',
}

describe('createShipment', () => {
  it('posts the input and returns the created shipment', async () => {
    mockFetch(sampleShipment, 201)
    const result = await api.createShipment(TOKEN, {
      sales_order_id: 'order-1', connection_id: 'conn-1', carrier: 1,
      recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
    })
    expect(result).toEqual(sampleShipment)
  })

  it('surfaces the backend error message on failure (no swallowed rejection)', async () => {
    mockFetchError('upstream_error', 'MyParcel rejected the shipment: invalid postal_code')
    await expect(api.createShipment(TOKEN, {
      sales_order_id: 'order-1', connection_id: 'conn-1', carrier: 1,
      recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
    })).rejects.toThrow('MyParcel rejected the shipment: invalid postal_code')
  })
})

describe('getShipmentLabel', () => {
  it('returns the raw response as a blob, not JSON-parsed', async () => {
    const blob = new Blob(['%PDF-fake'], { type: 'application/pdf' })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, blob: () => Promise.resolve(blob) }))

    const result = await api.getShipmentLabel(TOKEN, 's-1')

    expect(result).toBe(blob)
  })

  it('throws with the backend error message on failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false, status: 404,
      json: () => Promise.resolve({ data: null, error: { code: 'not_found', message: 'shipment not found' } }),
    }))

    await expect(api.getShipmentLabel(TOKEN, 'missing')).rejects.toThrow('shipment not found')
  })
})

describe('listShipmentsForOrder', () => {
  it('returns the shipments for an order', async () => {
    mockFetch([sampleShipment])
    const result = await api.listShipmentsForOrder(TOKEN, 'order-1')
    expect(result).toEqual([sampleShipment])
  })
})

describe('connectMyParcel', () => {
  it('posts the api key and returns the connection', async () => {
    mockFetch({ connection_id: 'conn-1', provider: 'myparcel', status: 'connected' }, 201)
    const result = await api.connectMyParcel(TOKEN, 'api-key-1')
    expect(result.connection_id).toBe('conn-1')
  })
})

describe('connectPostNL', () => {
  it('posts the PostNL account and returns the connection', async () => {
    mockFetch({ connection_id: 'conn-pn', provider: 'postnl', status: 'connected' }, 201)
    const result = await api.connectPostNL(TOKEN, { api_key: 'k', customer_code: 'SLTQ', customer_number: '10979997', sandbox: true })
    expect(result.connection_id).toBe('conn-pn')
    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(String(url)).toContain('/shipping/postnl/connect')
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({ api_key: 'k', customer_code: 'SLTQ', customer_number: '10979997', sandbox: true })
  })
})

describe('lookupAddress', () => {
  it('returns the street/city on a match', async () => {
    mockFetch({ street: 'Damrak', city: 'Amsterdam' })
    const result = await api.lookupAddress(TOKEN, '1012AB', '1')
    expect(result).toEqual({ street: 'Damrak', city: 'Amsterdam' })
  })

  it('returns null (not a thrown error) on a 404', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 404, json: () => Promise.resolve({ data: null, error: null }) }))
    const result = await api.lookupAddress(TOKEN, '9999ZZ', '999')
    expect(result).toBeNull()
  })

  it('throws on an unexpected failure', async () => {
    mockFetchError('upstream_error', 'pdok: request failed (500)', 502)
    await expect(api.lookupAddress(TOKEN, '1012AB', '1')).rejects.toThrow('pdok: request failed (500)')
  })
})

const shippingConnections = {
  connections: [{ id: 'conn-1', provider: 'myparcel', status: 'active', created_at: '2026-01-01T00:00:00Z', is_default: true }],
  default_connection_id: 'conn-1',
  configured_connection_id: null,
}

describe('listShippingConnections', () => {
  it('gets the shipping connections and the default', async () => {
    mockFetch(shippingConnections)
    const result = await api.listShippingConnections(TOKEN)
    expect(result).toEqual(shippingConnections)
    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(String(url)).toMatch(/\/shipping\/connections$/)
    expect(init?.method).toBe('GET')
  })
})

describe('updateShippingPreferences', () => {
  it('patches the default connection id', async () => {
    mockFetch(shippingConnections)
    await api.updateShippingPreferences(TOKEN, 'conn-1')
    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(String(url)).toMatch(/\/shipping\/preferences$/)
    expect(init?.method).toBe('PATCH')
    expect(JSON.parse(String(init?.body))).toEqual({ default_connection_id: 'conn-1' })
  })

  it('sends null to go back to automatic', async () => {
    mockFetch(shippingConnections)
    await api.updateShippingPreferences(TOKEN, null)
    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(String(init?.body))).toEqual({ default_connection_id: null })
  })

  it('throws the coded API error', async () => {
    mockFetchError('SHIPPING_PREFERENCES_FORBIDDEN', 'only an owner or admin can change the default shipping provider', 403)
    await expect(api.updateShippingPreferences(TOKEN, 'conn-1')).rejects.toMatchObject({ code: 'SHIPPING_PREFERENCES_FORBIDDEN' })
  })
})
