// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import { CodedApiError } from '../../api/codedApiError'
import type { DeadLetterPage } from '../types'

const TOKEN = 't'

function mockFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status >= 200 && status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const page: DeadLetterPage = { items: [], next_cursor: null, total: 0 }

beforeEach(() => { vi.restoreAllMocks() })

describe('dead letters api', () => {
  it('lists with only the filters that are set', async () => {
    const fetchMock = mockFetch({ data: page, error: null })

    await expect(api.listDeadLetters(TOKEN, { stream: 'events', consumer: 'c1', cursor: 7 })).resolves.toEqual(page)

    const url = new URL(fetchMock.mock.calls[0][0] as string, 'http://x')
    expect(url.pathname).toMatch(/\/admin\/dead-letters$/)
    expect(Object.fromEntries(url.searchParams)).toEqual({ stream: 'events', consumer: 'c1', cursor: '7' })
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer t')
  })

  it('retries with POST and discards with DELETE', async () => {
    const fetchMock = mockFetch({ data: null, error: null })

    await api.retryDeadLetter(TOKEN, 'webhooks', 9)
    await api.discardDeadLetter(TOKEN, 'events', 7)

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/admin\/dead-letters\/webhooks\/9\/retry$/)
    expect(fetchMock.mock.calls[0][1].method).toBe('POST')
    expect(fetchMock.mock.calls[1][0]).toMatch(/\/admin\/dead-letters\/events\/7$/)
    expect(fetchMock.mock.calls[1][1].method).toBe('DELETE')
  })

  it('throws the coded error from the envelope', async () => {
    mockFetch({ data: null, error: { code: 'DEAD_LETTER_RETRY_FAILED', message: 'dead letter could not be retried', detail: 'nats: timeout' } }, 502)

    const err = await api.retryDeadLetter(TOKEN, 'events', 7).catch(e => e)

    expect(err).toBeInstanceOf(CodedApiError)
    expect(err.code).toBe('DEAD_LETTER_RETRY_FAILED')
    expect(err.detail).toBe('nats: timeout')
  })
})
