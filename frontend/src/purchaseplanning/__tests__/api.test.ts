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

describe('listSeasonEvents', () => {
  it('lists next occurrences, or a given year', async () => {
    const fetchMock = mockFetch({ data: [], error: null })

    await api.listSeasonEvents('tok')
    await api.listSeasonEvents('tok', 2027)

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/procurement\/season-events$/)
    expect(fetchMock.mock.calls[1][0]).toMatch(/\/procurement\/season-events\?year=2027$/)
  })
})

describe('getSeasonPlan', () => {
  const plan = { as_of: '2026-10-06', year: 2026, window: { start: '2026-11-13', end: '2026-12-27' }, last_year_window: { start: '2025-11-13', end: '2025-12-27' }, safety_days: 7, suppliers: [] }

  it('asks for an event season', async () => {
    const fetchMock = mockFetch({ data: plan, error: null })

    const out = await api.getSeasonPlan('tok', { event: 'christmas', year: 2026, safety_days: 7, all_products: true, supplier_id: 'sup-1' })

    expect(out).toEqual(plan)
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/season-plan\?event=christmas&year=2026&safety_days=7&supplier_id=sup-1&all_products=true$/)
  })

  it('asks for a custom range', async () => {
    const fetchMock = mockFetch({ data: plan, error: null })

    await api.getSeasonPlan('tok', { from: '2026-11-01', to: '2026-11-11', safety_days: 0, all_products: false })

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/season-plan\?from=2026-11-01&to=2026-11-11&safety_days=0$/)
  })
})
