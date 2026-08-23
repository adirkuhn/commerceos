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
