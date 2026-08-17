// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Warehouse, Batch, StockMovement, StockLevel, TransferStockResult, ExpiringBatch, APIResponse } from '../types'

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

const sampleBatch: Batch = {
  id: 'batch-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'wh-1',
  lot_number: 'LOT-1', expiry_date: '2026-12-01', received_at: '2026-07-26T00:00:00Z',
  created_at: '2026-07-26T00:00:00Z', updated_at: '2026-07-26T00:00:00Z',
  quantity: 0, is_general_ledger: false,
}

const sampleMovement: StockMovement = {
  id: 'mv-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: null,
  type: 'adjustment', quantity_delta: -3, reason: '', occurred_at: '2026-07-26T00:00:00Z', created_at: '2026-07-26T00:00:00Z',
}

describe('listBatches', () => {
  it('calls correct endpoint', async () => {
    mockFetch([sampleBatch])
    const result = await api.listBatches(TOKEN, 'sku-1')
    expect(result).toEqual([sampleBatch])
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/batches',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('createBatch', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleBatch, 201)
    const input = { warehouse_id: 'wh-1', lot_number: 'LOT-1', expiry_date: '2026-12-01' }
    const result = await api.createBatch(TOKEN, 'sku-1', input)
    expect(result).toEqual(sampleBatch)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/batches')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'warehouse_id is required', 422)
    await expect(api.createBatch(TOKEN, 'sku-1', { warehouse_id: '' })).rejects.toThrow('warehouse_id is required')
  })
})

describe('updateBatch', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...sampleBatch, lot_number: 'LOT-2' })
    const input = { warehouse_id: 'wh-1', lot_number: 'LOT-2', expiry_date: null }
    const result = await api.updateBatch(TOKEN, 'batch-1', input)
    expect(result.lot_number).toBe('LOT-2')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/batches/batch-1')
    expect(opts.method).toBe('PATCH')
  })
})

describe('deleteBatch', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteBatch(TOKEN, 'batch-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/batches/batch-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on conflict when batch has movements', async () => {
    mockFetchError('BATCH_HAS_MOVEMENTS', 'batch has movements', 409)
    await expect(api.deleteBatch(TOKEN, 'batch-1')).rejects.toThrow('batch has movements')
  })
})

describe('getSKUStockLevel', () => {
  it('calls correct endpoint', async () => {
    const level: StockLevel = { sku_id: 'sku-1', total: 20, syncable: 15 }
    mockFetch(level)
    const result = await api.getSKUStockLevel(TOKEN, 'sku-1')
    expect(result).toEqual(level)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/stock-level',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('recordStockMovement', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleMovement, 201)
    const input = { warehouse_id: 'wh-1', type: 'adjustment' as const, quantity_delta: -3 }
    const result = await api.recordStockMovement(TOKEN, 'sku-1', input)
    expect(result).toEqual(sampleMovement)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/movements')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on invalid quantity', async () => {
    mockFetchError('VALIDATION_ERROR', 'quantity_delta must be negative for this type', 422)
    await expect(api.recordStockMovement(TOKEN, 'sku-1', { warehouse_id: 'wh-1', type: 'sale', quantity_delta: 3 }))
      .rejects.toThrow('quantity_delta must be negative for this type')
  })
})

describe('listStockMovements', () => {
  it('calls correct endpoint', async () => {
    mockFetch([sampleMovement])
    const result = await api.listStockMovements(TOKEN, 'sku-1')
    expect(result).toEqual([sampleMovement])
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/movements',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('recordRecount', () => {
  it('returns the movement when the count changed', async () => {
    mockFetch(sampleMovement)
    const result = await api.recordRecount(TOKEN, 'sku-1', { warehouse_id: 'wh-1', counted_quantity: 17 })
    expect(result).toEqual(sampleMovement)
  })

  it('returns null when nothing changed', async () => {
    mockFetch(null)
    const result = await api.recordRecount(TOKEN, 'sku-1', { warehouse_id: 'wh-1', counted_quantity: 20 })
    expect(result).toBeNull()
  })
})

describe('transferStock', () => {
  it('sends the input in body to the correct endpoint', async () => {
    const result: TransferStockResult = { out: sampleMovement, in: { ...sampleMovement, id: 'mv-2', quantity_delta: 3 } }
    mockFetch(result, 201)
    const input = { quantity: 3, from_warehouse_id: 'wh-1', to_warehouse_id: 'wh-2' }
    const got = await api.transferStock(TOKEN, 'sku-1', input)
    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/transfer')
    expect(opts.method).toBe('POST')
  })
})

describe('markDamagedOrLost', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleMovement, 201)
    const input = { warehouse_id: 'wh-1', quantity: 3, type: 'damaged' as const }
    const result = await api.markDamagedOrLost(TOKEN, 'sku-1', input)
    expect(result).toEqual(sampleMovement)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/mark-damaged-lost')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })
})

describe('receiveStock', () => {
  it('sends the input in body to the correct endpoint', async () => {
    const result = { batch: { ...sampleBatch, id: 'batch-1' }, movement: sampleMovement }
    mockFetch(result, 201)
    const input = { warehouse_id: 'wh-1', quantity: 8, lot_number: 'LOT-1' }
    const got = await api.receiveStock(TOKEN, 'sku-1', input)
    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/receive')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'quantity must be positive')
    await expect(api.receiveStock(TOKEN, 'sku-1', { warehouse_id: 'wh-1', quantity: 0 })).rejects.toThrow('quantity must be positive')
  })
})

describe('listExpiringBatches', () => {
  it('calls correct endpoint with default within_days', async () => {
    const batches: ExpiringBatch[] = [{
      batch_id: 'batch-1', sku_id: 'sku-1', sku_code: 'SKU-1', product_name: 'Widget',
      warehouse_id: 'wh-1', warehouse_name: 'Main DC', lot_number: 'LOT-1',
      expiry_date: '2026-08-10T00:00:00Z', quantity: 10,
    }]
    mockFetch(batches)
    const result = await api.listExpiringBatches(TOKEN)
    expect(result).toEqual(batches)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/batches/expiring-soon?within_days=30',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('passes a custom within_days', async () => {
    mockFetch([])
    await api.listExpiringBatches(TOKEN, 7)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/batches/expiring-soon?within_days=7',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('discardBatch', () => {
  it('sends a multipart form with the fields and photo, and does not force a Content-Type header', async () => {
    mockFetch({
      id: 'discard-1', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1', batch_id: 'batch-1',
      movement_id: 'move-1', quantity: 3, reason: 'expired', discarded_by: 'user-1',
      discarded_at: '2026-08-17T00:00:00Z', created_at: '2026-08-17T00:00:00Z',
    })
    const photo = new File(['fake-bytes'], 'photo.jpg', { type: 'image/jpeg' })

    const result = await api.discardBatch(TOKEN, 'sku-1', {
      warehouse_id: 'wh-1', batch_id: 'batch-1', quantity: 3, reason: 'expired', photo,
    })

    expect(result.id).toBe('discard-1')
    const call = vi.mocked(fetch).mock.calls[0]
    expect(call[0]).toBe('https://api.commerceos.dev.localhost/api/v1/inventory/skus/sku-1/discard')
    const options = call[1] as RequestInit
    expect(options.method).toBe('POST')
    expect(options.headers).not.toHaveProperty('Content-Type')
    const form = options.body as FormData
    expect(form.get('warehouse_id')).toBe('wh-1')
    expect(form.get('batch_id')).toBe('batch-1')
    expect(form.get('quantity')).toBe('3')
    expect(form.get('reason')).toBe('expired')
    expect(form.get('photo')).toBe(photo)
  })

  it('omits batch_id and photo when discarding general-ledger stock with no photo', async () => {
    mockFetch({
      id: 'discard-2', tenant_id: 't1', sku_id: 'sku-1', warehouse_id: 'wh-1',
      movement_id: 'move-2', quantity: 2, reason: '', discarded_by: '',
      discarded_at: '2026-08-17T00:00:00Z', created_at: '2026-08-17T00:00:00Z',
    })

    await api.discardBatch(TOKEN, 'sku-1', { warehouse_id: 'wh-1', quantity: 2 })

    const form = (vi.mocked(fetch).mock.calls[0][1] as RequestInit).body as FormData
    expect(form.get('batch_id')).toBeNull()
    expect(form.get('photo')).toBeNull()
  })

  it('surfaces the server error message on failure', async () => {
    mockFetchError('VALIDATION_ERROR', 'insufficient stock for this batch/ledger')

    await expect(api.discardBatch(TOKEN, 'sku-1', { warehouse_id: 'wh-1', quantity: 999 }))
      .rejects.toThrow('insufficient stock for this batch/ledger')
  })
})

describe('listDiscards', () => {
  it('calls the endpoint with no query params by default and returns discards + total', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200,
      json: () => Promise.resolve({ data: [], total: 0, error: null }),
    }))

    const result = await api.listDiscards(TOKEN)

    expect(result).toEqual({ discards: [], total: 0 })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/inventory/discards',
      expect.objectContaining({ headers: expect.objectContaining({ Authorization: 'Bearer test-token' }) }),
    )
  })

  it('passes warehouse/sku/date/pagination filters as query params', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200,
      json: () => Promise.resolve({ data: [], total: 0, error: null }),
    }))

    await api.listDiscards(TOKEN, { warehouseId: 'wh-1', skuId: 'sku-1', from: '2026-08-01', to: '2026-08-31', limit: 10, offset: 20 })

    const url = vi.mocked(fetch).mock.calls[0][0] as string
    expect(url).toContain('warehouse_id=wh-1')
    expect(url).toContain('sku_id=sku-1')
    expect(url).toContain('from=2026-08-01')
    expect(url).toContain('to=2026-08-31')
    expect(url).toContain('limit=10')
    expect(url).toContain('offset=20')
  })
})
