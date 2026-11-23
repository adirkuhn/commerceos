// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import { CodedApiError } from '../../api/codedApiError'
import type { AssistantRequest } from '../types'

function mockFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status >= 200 && status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const req: AssistantRequest = {
  id: 'r-1', title: 'hi', status: 'queued', attempts: 0, next_attempt_at: '2026-11-23T12:00:00Z',
  user_id: 'u-1', created_at: '2026-11-23T12:00:00Z', updated_at: '2026-11-23T12:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('assistant api', () => {
  it('lists with only the query params that are set', async () => {
    const fetchMock = mockFetch({ data: { requests: [req], next_cursor: null }, error: null })

    await expect(api.listAssistantRequests('t', { status: 'queued', cursor: 'c-1' })).resolves.toEqual({ requests: [req], next_cursor: null })

    const url = new URL(fetchMock.mock.calls[0][0] as string, 'http://x')
    expect(url.pathname).toMatch(/\/assistant\/requests$/)
    expect(Object.fromEntries(url.searchParams)).toEqual({ status: 'queued', cursor: 'c-1' })
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer t')
  })

  it('lists without a query string when nothing is set', async () => {
    const fetchMock = mockFetch({ data: { requests: [] }, error: null })
    await api.listAssistantRequests('t')
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/assistant\/requests$/)
  })

  it('creates and replies with the text body, cancels with POST', async () => {
    const fetchMock = mockFetch({ data: req, error: null })

    await api.createAssistantRequest('t', 'status CO-12?')
    await api.replyToAssistantRequest('t', 'r-1', 'Ana Bakery')
    await api.cancelAssistantRequest('t', 'r-1')

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/assistant\/requests$/)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST', body: JSON.stringify({ text: 'status CO-12?' }) })
    expect(fetchMock.mock.calls[1][0]).toMatch(/\/assistant\/requests\/r-1\/reply$/)
    expect(fetchMock.mock.calls[1][1].body).toBe(JSON.stringify({ text: 'Ana Bakery' }))
    expect(fetchMock.mock.calls[2][0]).toMatch(/\/assistant\/requests\/r-1\/cancel$/)
    expect(fetchMock.mock.calls[2][1].method).toBe('POST')
  })

  it('gets one request with its conversation', async () => {
    const fetchMock = mockFetch({ data: { request: req, messages: [] }, error: null })
    await expect(api.getAssistantRequest('t', 'r-1')).resolves.toEqual({ request: req, messages: [] })
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/assistant\/requests\/r-1$/)
  })

  it('throws the coded API error', async () => {
    mockFetch({ data: null, error: { code: 'ASSISTANT_NOT_CANCELLABLE', message: 'nope' } }, 409)
    const err = await api.cancelAssistantRequest('t', 'r-1').catch(e => e)
    expect(err).toBeInstanceOf(CodedApiError)
    expect(err.code).toBe('ASSISTANT_NOT_CANCELLABLE')
  })

  it('confirms a proposal with the chosen warehouse and rejects one, both returning the detail', async () => {
    const detail = { request: req, messages: [], proposals: [] }
    const fetchMock = mockFetch({ data: detail, error: null })

    await expect(api.confirmAssistantProposal('t', 'r-1', 'p-1', { warehouse_id: 'w-2' })).resolves.toEqual(detail)
    await api.confirmAssistantProposal('t', 'r-1', 'p-1')
    await api.rejectAssistantProposal('t', 'r-1', 'p-1')

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/assistant\/requests\/r-1\/proposals\/p-1\/confirm$/)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST', body: JSON.stringify({ warehouse_id: 'w-2' }) })
    expect(fetchMock.mock.calls[1][1].body).toBe('{}')
    expect(fetchMock.mock.calls[2][0]).toMatch(/\/assistant\/requests\/r-1\/proposals\/p-1\/reject$/)
    expect(fetchMock.mock.calls[2][1].method).toBe('POST')
  })
})
