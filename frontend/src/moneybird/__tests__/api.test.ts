import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import { CodedApiError } from '../../api/codedApiError'

const TOKEN = 'tok'
const BASE = `${import.meta.env.VITE_API_BASE_URL}/integrations/moneybird`

function mockFetch(data: unknown, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300, status, json: () => Promise.resolve({ data, error: null }),
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

beforeEach(() => { vi.restoreAllMocks() })

describe('moneybird api', () => {
  it('connects with the token', async () => {
    const fetchMock = mockFetch({ connection_id: 'conn-1' }, 201)
    await expect(api.connectMoneybird(TOKEN, 'mb-token')).resolves.toEqual({ connection_id: 'conn-1' })
    expect(fetchMock).toHaveBeenCalledWith(`${BASE}/connect`, expect.objectContaining({ method: 'POST', body: JSON.stringify({ api_token: 'mb-token' }) }))
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer tok')
  })

  it('reads the setup', async () => {
    const fetchMock = mockFetch({ settings: {}, administrations: [], tax_rates: [], ledger_accounts: [], payment_methods: [] })
    await api.getMoneybirdSetup(TOKEN)
    expect(fetchMock).toHaveBeenCalledWith(BASE, expect.objectContaining({ method: 'GET' }))
  })

  it('patches only the fields given', async () => {
    const fetchMock = mockFetch({})
    await api.updateMoneybirdSettings(TOKEN, { auto_sync: true })
    expect(fetchMock).toHaveBeenCalledWith(`${BASE}/settings`, expect.objectContaining({ method: 'PATCH', body: '{"auto_sync":true}' }))
  })

  it('sends a sales order and a quote', async () => {
    const fetchMock = mockFetch({})
    await api.sendToMoneybird(TOKEN, 'sales_order', 'so-1')
    await api.sendToMoneybird(TOKEN, 'quote', 'q-1')
    expect(fetchMock.mock.calls[0][0]).toBe(`${BASE}/sales-orders/so-1/send`)
    expect(fetchMock.mock.calls[1][0]).toBe(`${BASE}/quotes/q-1/send`)
    expect(fetchMock.mock.calls[0][1].method).toBe('POST')
  })

  it('reads a document state', async () => {
    const fetchMock = mockFetch({ document: null, administration_id: 'adm', auto_sync: false })
    await api.getMoneybirdDocument(TOKEN, 'quote', 'q-1')
    expect(fetchMock.mock.calls[0][0]).toBe(`${BASE}/quotes/q-1`)
  })

  it('throws the coded error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false, status: 409, json: () => Promise.resolve({ data: null, error: { code: 'MONEYBIRD_DUPLICATE_REFERENCE', message: 'dup' } }),
    }))
    const err = await api.sendToMoneybird(TOKEN, 'sales_order', 'so-1').catch(e => e)
    expect(err).toBeInstanceOf(CodedApiError)
    expect(err.code).toBe('MONEYBIRD_DUPLICATE_REFERENCE')
  })
})
