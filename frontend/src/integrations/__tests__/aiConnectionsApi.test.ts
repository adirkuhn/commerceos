import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { AIConnection, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const conn: AIConnection = {
  id: 'ai-1', provider: 'nvidia', position: 0, label: 'key 1', enabled: true, has_api_key: true,
  status: 'connected', created_at: '2026-11-23T00:00:00Z',
}

function call(fetchMock: ReturnType<typeof vi.fn>): { url: string; init: RequestInit; body: unknown } {
  const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
  return { url, init, body: init.body ? JSON.parse(init.body as string) : undefined }
}

beforeEach(() => { vi.restoreAllMocks() })

describe('AI connections api', () => {
  it('lists in routing order', async () => {
    const f = mockFetch([conn])
    expect(await api.listAIConnections(TOKEN)).toEqual([conn])
    const { url, init } = call(f)
    expect(url).toMatch(/\/integrations\/ai-connections$/)
    expect(init.method).toBe('GET')
  })

  it('creates, dropping empty optional fields', async () => {
    const f = mockFetch(conn)
    await api.createAIConnection(TOKEN, { provider: 'nvidia', apiKey: 'nvapi-x', baseUrl: '', model: '', fallbackModels: 'm2', label: 'key 1' })
    const { url, init, body } = call(f)
    expect(url).toMatch(/\/integrations\/ai-connections$/)
    expect(init.method).toBe('POST')
    expect(body).toEqual({ provider: 'nvidia', api_key: 'nvapi-x', fallback_models: 'm2', label: 'key 1' })
  })

  it('updates only the given fields and never sends an empty key', async () => {
    const f = mockFetch(conn)
    await api.updateAIConnection(TOKEN, 'ai-1', { enabled: false, apiKey: '', label: '' })
    const { url, init, body } = call(f)
    expect(url).toMatch(/\/integrations\/ai-connections\/ai-1$/)
    expect(init.method).toBe('PATCH')
    expect(body).toEqual({ enabled: false, label: '' })
  })

  it('sends a newly typed key', async () => {
    const f = mockFetch(conn)
    await api.updateAIConnection(TOKEN, 'ai-1', { apiKey: 'nvapi-new', model: 'm' })
    expect(call(f).body).toEqual({ api_key: 'nvapi-new', model: 'm' })
  })

  it('deletes', async () => {
    const f = mockFetch(null)
    await api.deleteAIConnection(TOKEN, 'ai-1')
    const { url, init } = call(f)
    expect(url).toMatch(/\/integrations\/ai-connections\/ai-1$/)
    expect(init.method).toBe('DELETE')
  })

  it('reorders', async () => {
    const f = mockFetch([conn])
    await api.reorderAIConnections(TOKEN, ['ai-2', 'ai-1'])
    const { url, init, body } = call(f)
    expect(url).toMatch(/\/integrations\/ai-connections\/order$/)
    expect(init.method).toBe('PUT')
    expect(body).toEqual({ ids: ['ai-2', 'ai-1'] })
  })
})
