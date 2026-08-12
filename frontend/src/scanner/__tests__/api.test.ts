import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Booking, BookingDetail, ScannerPairingCode, APIResponse } from '../types'

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
