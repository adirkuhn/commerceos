// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { acknowledgeBatchExpiry, listCloseToExpiry, unacknowledgeBatchExpiry } from '../api'
import { CodedApiError } from '../../../api/codedApiError'

const BASE = `${import.meta.env.VITE_API_BASE_URL}/inventory/batches`

function stubFetch(status: number, body?: unknown) {
  const fn = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300, status,
    json: () => (body === undefined ? Promise.reject(new SyntaxError('no body')) : Promise.resolve(body)),
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listCloseToExpiry', () => {
  it('sends the window, list and page, and unwraps data', async () => {
    const data = { today: '2026-10-12', items: [] }
    const fetchMock = stubFetch(200, { data, error: null })

    await expect(listCloseToExpiry('tok', { window: 30, acknowledged: true, page: 2, perPage: 25 })).resolves.toEqual(data)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe(`${BASE}/close-to-expiry?window=30&acknowledged=true&page=2&per_page=25`)
    expect(init.headers.Authorization).toBe('Bearer tok')
  })

  it('leaves defaults out of the query', async () => {
    const fetchMock = stubFetch(200, { data: {}, error: null })
    await listCloseToExpiry('tok', {})
    expect(fetchMock.mock.calls[0][0]).toBe(`${BASE}/close-to-expiry`)
  })

  it('throws the coded API error', async () => {
    stubFetch(422, { data: null, error: { code: 'EXPIRY_WINDOW_UNKNOWN', message: 'unknown' } })
    await expect(listCloseToExpiry('tok', { window: 7 })).rejects.toBeInstanceOf(CodedApiError)
  })
})

describe('acknowledgeBatchExpiry', () => {
  it('posts the window and accepts an empty 204', async () => {
    const fetchMock = stubFetch(204)
    await expect(acknowledgeBatchExpiry('tok', 'b-1', 30)).resolves.toBeUndefined()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe(`${BASE}/b-1/expiry-acknowledgements`)
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ window_days: 30 })
  })

  it('throws the coded API error', async () => {
    stubFetch(422, { data: null, error: { code: 'BATCH_NOT_IN_EXPIRY_WINDOW', message: 'moved' } })
    await expect(acknowledgeBatchExpiry('tok', 'b-1', 30)).rejects.toMatchObject({ code: 'BATCH_NOT_IN_EXPIRY_WINDOW' })
  })
})

describe('unacknowledgeBatchExpiry', () => {
  it('deletes the window acknowledgement', async () => {
    const fetchMock = stubFetch(204)
    await unacknowledgeBatchExpiry('tok', 'b-1', 30)
    expect(fetchMock.mock.calls[0][0]).toBe(`${BASE}/b-1/expiry-acknowledgements/30`)
    expect(fetchMock.mock.calls[0][1].method).toBe('DELETE')
  })
})
