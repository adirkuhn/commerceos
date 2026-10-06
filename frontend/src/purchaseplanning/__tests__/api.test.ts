// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import * as api from '../api'
import type { APIResponse, ReorderRecommendations } from '../types'

const sample: ReorderRecommendations = { as_of: '2026-10-06', coverage_days: 30, safety_days: 7, suppliers: [] }

function mockFetch<T>(body: APIResponse<T>, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('getReorderRecommendations', () => {
  it('sends the planning window as query params with the bearer token', async () => {
    const fetchMock = mockFetch({ data: sample, error: null })

    const out = await api.getReorderRecommendations('tok', { coverage_days: 60, safety_days: 0, supplier_id: 'sup-1' })

    expect(out).toEqual(sample)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/procurement\/recommendations\?coverage_days=60&safety_days=0&supplier_id=sup-1$/)
    expect(init.headers.Authorization).toBe('Bearer tok')
  })

  it('leaves supplier_id out when not filtering', async () => {
    const fetchMock = mockFetch({ data: sample, error: null })

    await api.getReorderRecommendations('tok', { coverage_days: 30, safety_days: 7 })

    expect(fetchMock.mock.calls[0][0]).toMatch(/\?coverage_days=30&safety_days=7$/)
  })

  it('throws the API error message', async () => {
    mockFetch({ data: null, error: { code: 'INVALID_REQUEST', message: 'coverage_days must be 1-365' } }, 422)

    await expect(api.getReorderRecommendations('tok', { coverage_days: 0, safety_days: 7 }))
      .rejects.toThrow('coverage_days must be 1-365')
  })
})
