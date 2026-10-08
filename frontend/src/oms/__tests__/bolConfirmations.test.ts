// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { listBolShipmentConfirmations, latestBolConfirmation, isBolConfirmationPending } from '../bolConfirmations'
import type { BolShipmentConfirmation } from '../bolConfirmations'

function confirmation(overrides: Partial<BolShipmentConfirmation> = {}): BolShipmentConfirmation {
  return {
    id: 'conf-1', shipment_id: 'ship-1', external_order_id: 'C000HLRF87', action: 'create_shipment', transport_id: '', transporter_code: 'TNT',
    track_and_trace: '3SMYPA859699114', process_status_id: 'ps-1', status: 'pending', error_message: '',
    needs_manual_action: false, created_at: '2026-10-07T14:11:00Z', resolved_at: null,
    ...overrides,
  }
}

afterEach(() => vi.unstubAllGlobals())

describe('listBolShipmentConfirmations', () => {
  it('GETs the order\'s confirmations with the bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve({ data: [confirmation()], error: null }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await listBolShipmentConfirmations('tok', 'so-1')

    expect(result).toEqual([confirmation()])
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/sales-orders\/so-1\/bol-shipment-confirmations$/)
    expect(init.headers.Authorization).toBe('Bearer tok')
  })

  it('throws the API error message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false, status: 404, json: () => Promise.resolve({ data: null, error: { code: 'NOT_FOUND', message: 'sales order not found' } }),
    }))

    await expect(listBolShipmentConfirmations('tok', 'missing')).rejects.toThrow('sales order not found')
  })
})

describe('latestBolConfirmation', () => {
  it('is the first (newest) attempt, or null when there is none', () => {
    expect(latestBolConfirmation([confirmation({ id: 'new' }), confirmation({ id: 'old' })])?.id).toBe('new')
    expect(latestBolConfirmation([])).toBeNull()
  })
})

describe('isBolConfirmationPending', () => {
  it('is true only while Bol is still processing the newest attempt', () => {
    expect(isBolConfirmationPending([confirmation({ status: 'pending' })])).toBe(true)
    expect(isBolConfirmationPending([confirmation({ status: 'success' })])).toBe(false)
    expect(isBolConfirmationPending([confirmation({ status: 'failure' }), confirmation({ status: 'pending' })])).toBe(false)
    expect(isBolConfirmationPending([])).toBe(false)
  })
})
