import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as api from '../api'
import type { Booking, BookingDetail, ScannerPairingCode, APIResponse, PickingSession } from '../types'

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

beforeEach(() => { vi.restoreAllMocks() })

const samplePairing: ScannerPairingCode = {
  id: 'pairing-1', tenant_id: 'tenant-1', code: '123456', token: 'opaque-token-abc', status: 'pending',
  expires_at: '2026-08-05T00:05:00Z', created_at: '2026-08-05T00:00:00Z',
}

const sampleBooking: Booking = {
  id: 'booking-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1',
  status: 'in_progress', started_at: '2026-08-05T00:00:00Z', completed_by: '',
}

describe('claimPairingCode', () => {
  it('posts the code with no Authorization header', async () => {
    mockFetch({ token: 'signed-jwt' })
    const result = await api.claimPairingCode('123456')

    expect(result).toEqual({ token: 'signed-jwt' })
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/iam/scanner-pairing/claim')
    expect(JSON.parse(opts.body as string)).toEqual({ code: '123456' })
    expect((opts.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('throws on invalid code', async () => {
    mockFetchError('VALIDATION_ERROR', 'pairing code is invalid or expired')
    await expect(api.claimPairingCode('000000')).rejects.toThrow('pairing code is invalid or expired')
  })
})

describe('claimPairingCodeByToken', () => {
  it('posts the token with no Authorization header', async () => {
    mockFetch({ token: 'signed-jwt' })
    const result = await api.claimPairingCodeByToken('opaque-abc')

    expect(result).toEqual({ token: 'signed-jwt' })
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/iam/scanner-pairing/claim-token')
    expect(JSON.parse(opts.body as string)).toEqual({ token: 'opaque-abc' })
    expect((opts.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('throws on invalid token', async () => {
    mockFetchError('VALIDATION_ERROR', 'pairing code is invalid or expired')
    await expect(api.claimPairingCodeByToken('does-not-exist')).rejects.toThrow('pairing code is invalid or expired')
  })
})

describe('generatePairingCode', () => {
  it('sends the caller token to the correct endpoint', async () => {
    mockFetch(samplePairing, 201)
    const result = await api.generatePairingCode(TOKEN)

    expect(result).toEqual(samplePairing)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/iam/scanner-pairing')
    expect(opts.method).toBe('POST')
    expect((opts.headers as Record<string, string>).Authorization).toBe(`Bearer ${TOKEN}`)
  })
})

describe('getPairingCodeStatus', () => {
  it('calls the correct endpoint', async () => {
    mockFetch({ ...samplePairing, status: 'claimed' })
    const result = await api.getPairingCodeStatus(TOKEN, 'pairing-1')

    expect(result.status).toBe('claimed')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/iam/scanner-pairing/pairing-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('refreshScannerSession', () => {
  it('posts to the correct endpoint with the caller token', async () => {
    mockFetch({ token: 'new-signed-jwt' })
    const result = await api.refreshScannerSession(TOKEN)

    expect(result).toEqual({ token: 'new-signed-jwt' })
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/iam/scanner-pairing/refresh')
    expect(opts.method).toBe('POST')
    expect((opts.headers as Record<string, string>).Authorization).toBe(`Bearer ${TOKEN}`)
  })
})

describe('onTokenActivity', () => {
  afterEach(() => api.setOnTokenActivity(null))

  it('is invoked with the token after a successful authenticated call', async () => {
    mockFetch(samplePairing, 201)
    const handler = vi.fn()
    api.setOnTokenActivity(handler)

    await api.generatePairingCode(TOKEN)

    expect(handler).toHaveBeenCalledWith(TOKEN)
  })

  it('is not invoked for the unauthenticated claim call', async () => {
    mockFetch({ token: 'signed-jwt' })
    const handler = vi.fn()
    api.setOnTokenActivity(handler)

    await api.claimPairingCode('123456')

    expect(handler).not.toHaveBeenCalled()
  })

  it('is not invoked when the call fails', async () => {
    mockFetchError('VALIDATION_ERROR', 'nope')
    const handler = vi.fn()
    api.setOnTokenActivity(handler)

    await expect(api.getPairingCodeStatus(TOKEN, 'pairing-1')).rejects.toThrow()

    expect(handler).not.toHaveBeenCalled()
  })
})

describe('startOrResumeBooking', () => {
  it('posts to the correct endpoint with no body when the PO already has a warehouse', async () => {
    mockFetch(sampleBooking)
    const result = await api.startOrResumeBooking(TOKEN, 'po-1')

    expect(result).toEqual(sampleBooking)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1/booking')
    expect(opts.method).toBe('POST')
    expect(opts.body).toBeUndefined()
  })

  it('sends warehouse_id when provided', async () => {
    mockFetch(sampleBooking)
    await api.startOrResumeBooking(TOKEN, 'po-1', 'warehouse-1')

    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string)).toEqual({ warehouse_id: 'warehouse-1' })
  })
})

describe('getBooking', () => {
  it('returns booking detail', async () => {
    const detail: BookingDetail = { booking: sampleBooking, line_items: [], scans: [] }
    mockFetch(detail)
    const result = await api.getBooking(TOKEN, 'booking-1')

    expect(result).toEqual(detail)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/bookings/booking-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('scanBookingItem', () => {
  it('sends the scan input to the correct endpoint', async () => {
    const result = { scan: { id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', quantity: 3, scanned_at: '', scanned_by: '' } }
    mockFetch(result, 201)
    const got = await api.scanBookingItem(TOKEN, 'booking-1', { code: 'BARCODE-1', quantity: 3 })

    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/bookings/booking-1/scan')
    expect(JSON.parse(opts.body as string)).toEqual({ code: 'BARCODE-1', quantity: 3 })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'no product matches this barcode')
    await expect(api.scanBookingItem(TOKEN, 'booking-1', { code: 'UNKNOWN', quantity: 1 })).rejects.toThrow('no product matches this barcode')
  })
})

describe('resolveBookingScanCode', () => {
  it('sends the code to the correct endpoint', async () => {
    const result = { sku_id: 'sku-1', product_name: 'Widget', units_per_scan: 12 }
    mockFetch(result)
    const got = await api.resolveBookingScanCode(TOKEN, 'booking-1', 'CASE-1')

    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/bookings/booking-1/resolve-scan')
    expect(JSON.parse(opts.body as string)).toEqual({ code: 'CASE-1' })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'no product matches this barcode')
    await expect(api.resolveBookingScanCode(TOKEN, 'booking-1', 'UNKNOWN')).rejects.toThrow('no product matches this barcode')
  })
})

describe('pauseBooking', () => {
  it('posts to the correct endpoint', async () => {
    mockFetch({ ...sampleBooking, status: 'paused' })
    const result = await api.pauseBooking(TOKEN, 'booking-1')

    expect(result.status).toBe('paused')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/bookings/booking-1/pause',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('completeBooking', () => {
  it('posts to the correct endpoint', async () => {
    mockFetch({ booking: { ...sampleBooking, status: 'completed' }, missing_items: [] })
    const result = await api.completeBooking(TOKEN, 'booking-1')

    expect(result.booking.status).toBe('completed')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/bookings/booking-1/complete',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

const samplePickingSession: PickingSession = {
  id: 'session-1', sales_order_id: 'order-1', status: 'in_progress', started_at: '2026-08-05T00:00:00Z', completed_by: '',
}

describe('startOrResumePicking', () => {
  it('posts to the correct sales-order-scoped endpoint (no /oms prefix)', async () => {
    mockFetch(samplePickingSession)
    const result = await api.startOrResumePicking(TOKEN, 'order-1')

    expect(result).toEqual(samplePickingSession)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/sales-orders/order-1/picking',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('getPickingSession', () => {
  it('calls the correct endpoint', async () => {
    const detail = { session: samplePickingSession, line_items: [], scans: [] }
    mockFetch(detail)
    const result = await api.getPickingSession(TOKEN, 'session-1')

    expect(result).toEqual(detail)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('scanPickItem', () => {
  it('sends the scan input to the correct endpoint', async () => {
    const result = { scan: { id: 'scan-1', session_id: 'session-1', sku_id: 'sku-1', action_type: 'pick', quantity: 3, scanned_at: '', scanned_by: '' } }
    mockFetch(result, 201)
    const got = await api.scanPickItem(TOKEN, 'session-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 3, line_item_id: 'li-1' })

    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/scan')
    expect(JSON.parse(opts.body as string)).toEqual({ sku_id: 'sku-1', action_type: 'pick', quantity: 3, line_item_id: 'li-1' })
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'quantity must be positive')
    await expect(api.scanPickItem(TOKEN, 'session-1', { sku_id: 'sku-2', action_type: 'pick', quantity: 0 })).rejects.toThrow('quantity must be positive')
  })

  it('substitute is rejected -- use substituteLineItem instead', async () => {
    mockFetchError('VALIDATION_ERROR', 'substitute action must use the substitute-batch endpoint')
    await expect(api.scanPickItem(TOKEN, 'session-1', { sku_id: 'sku-2', action_type: 'substitute' })).rejects.toThrow('substitute action must use the substitute-batch endpoint')
  })
})

describe('substituteLineItem', () => {
  it('sends the batch of substitute lines to the correct endpoint', async () => {
    const result = {
      scans: [
        { id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-a', action_type: 'substitute', quantity: 3, original_quantity_covered: 1, return_to_stock: false, scanned_at: '', scanned_by: '' },
        { id: 'scan-2', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-b', action_type: 'substitute', quantity: 4, original_quantity_covered: 1, return_to_stock: true, scanned_at: '', scanned_by: '' },
      ],
    }
    mockFetch(result, 201)
    const input = {
      line_item_id: 'li-1',
      lines: [
        { sku_id: 'sku-a', substitute_quantity: 3, original_quantity_covered: 1, return_to_stock: false },
        { sku_id: 'sku-b', substitute_quantity: 4, original_quantity_covered: 1, return_to_stock: true },
      ],
    }
    const got = await api.substituteLineItem(TOKEN, 'session-1', input)

    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/substitute-batch')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on coverage mismatch validation error', async () => {
    mockFetchError('VALIDATION_ERROR', "substitute lines must cover exactly the original line item's quantity")
    await expect(api.substituteLineItem(TOKEN, 'session-1', {
      line_item_id: 'li-1', lines: [{ sku_id: 'sku-a', substitute_quantity: 1, original_quantity_covered: 1 }],
    })).rejects.toThrow("substitute lines must cover exactly the original line item's quantity")
  })
})

describe('refundLineItem', () => {
  it('sends the refund to the correct endpoint', async () => {
    const result = {
      scan: { id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'refund', quantity: 1, scanned_at: '', scanned_by: '' },
    }
    mockFetch(result, 201)
    const input = { line_item_id: 'li-1', quantity: 1, reason: 'damaged' }
    const got = await api.refundLineItem(TOKEN, 'session-1', input)

    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/refund')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on a permission-denied error', async () => {
    mockFetchError('FORBIDDEN', 'caller does not have permission to perform this action', 403)
    await expect(api.refundLineItem(TOKEN, 'session-1', { line_item_id: 'li-1', quantity: 1 }))
      .rejects.toThrow('caller does not have permission to perform this action')
  })
})

describe('resolvePickScanCode', () => {
  it('sends the code to the correct endpoint', async () => {
    const result = { sku_id: 'sku-1', product_name: 'Widget' }
    mockFetch(result)
    const got = await api.resolvePickScanCode(TOKEN, 'session-1', 'BARCODE-1')

    expect(got).toEqual(result)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/resolve-scan')
    expect(JSON.parse(opts.body as string)).toEqual({ code: 'BARCODE-1' })
  })
})

describe('deletePickScan', () => {
  it('deletes the correct scan', async () => {
    mockFetch(null)
    await api.deletePickScan(TOKEN, 'session-1', 'scan-1')

    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/scans/scan-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })
})

describe('pausePicking', () => {
  it('posts to the correct endpoint', async () => {
    mockFetch({ ...samplePickingSession, status: 'paused' })
    const result = await api.pausePicking(TOKEN, 'session-1')

    expect(result.status).toBe('paused')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/pause',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('completePicking', () => {
  it('posts to the correct endpoint', async () => {
    mockFetch({ completed: true, session: { ...samplePickingSession, status: 'completed' }, missing_items: [] })
    const result = await api.completePicking(TOKEN, 'session-1')

    expect(result.session?.status).toBe('completed')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/picking-sessions/session-1/complete',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ resolutions: undefined }) }),
    )
  })
})

describe('addSalesOrderLineItem', () => {
  it('posts to the sales-order line-items endpoint', async () => {
    const lineItem = { id: 'li-new', name: 'Freebie Widget', sku_code: 'FREE-1', barcode: '', quantity: 1, price_minor_units: 0, subtotal_minor_units: 0, subtotal_tax_minor_units: 0, total_minor_units: 0, total_tax_minor_units: 0 }
    mockFetch(lineItem, 201)
    const result = await api.addSalesOrderLineItem(TOKEN, 'order-1', { sku_id: 'sku-9', quantity: 1, price_minor_units: 0 })

    expect(result).toEqual(lineItem)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/sales-orders/order-1/line-items')
    expect(JSON.parse(opts.body as string)).toEqual({ sku_id: 'sku-9', quantity: 1, price_minor_units: 0 })
  })
})
