// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReorderRecommendations, ReorderRecommendationsQuery } from '../types'

vi.mock('../api', () => ({ getReorderRecommendations: vi.fn() }))
import * as api from '../api'
import { useReorderRecommendations } from '../hooks/useReorderRecommendations'

const result = (coverage: number): ReorderRecommendations => ({ as_of: '2026-10-06', coverage_days: coverage, safety_days: 7, suppliers: [] })

beforeEach(() => vi.clearAllMocks())

describe('useReorderRecommendations', () => {
  it('loads recommendations for the query', async () => {
    vi.mocked(api.getReorderRecommendations).mockResolvedValue(result(30))
    const query = { coverage_days: 30, safety_days: 7 }

    const { result: hook } = renderHook(() => useReorderRecommendations('tok', query))

    await waitFor(() => expect(hook.current.data).toEqual(result(30)))
    expect(hook.current.loading).toBe(false)
    expect(api.getReorderRecommendations).toHaveBeenCalledWith('tok', query)
  })

  it('surfaces errors', async () => {
    vi.mocked(api.getReorderRecommendations).mockRejectedValue(new Error('boom'))

    const { result: hook } = renderHook(() => useReorderRecommendations('tok', { coverage_days: 30, safety_days: 7 }))

    await waitFor(() => expect(hook.current.error).toBe('boom'))
    expect(hook.current.data).toBeNull()
  })

  it('ignores a slower answer to an older query', async () => {
    let resolveOld: (r: ReorderRecommendations) => void = () => {}
    vi.mocked(api.getReorderRecommendations)
      .mockImplementationOnce(() => new Promise(r => { resolveOld = r }))
      .mockResolvedValueOnce(result(60))

    const { result: hook, rerender } = renderHook(
      ({ q }: { q: ReorderRecommendationsQuery }) => useReorderRecommendations('tok', q),
      { initialProps: { q: { coverage_days: 30, safety_days: 7 } } },
    )
    rerender({ q: { coverage_days: 60, safety_days: 7 } })
    await waitFor(() => expect(hook.current.data?.coverage_days).toBe(60))

    resolveOld(result(30))
    await new Promise(r => setTimeout(r, 0))
    expect(hook.current.data?.coverage_days).toBe(60)
  })

  it('does nothing without a token', () => {
    renderHook(() => useReorderRecommendations('', { coverage_days: 30, safety_days: 7 }))
    expect(api.getReorderRecommendations).not.toHaveBeenCalled()
  })
})
