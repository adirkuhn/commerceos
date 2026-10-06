// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import type { SeasonPlan } from '../types'

vi.mock('../api', () => ({ getSeasonPlan: vi.fn(), listSeasonEvents: vi.fn() }))
import * as api from '../api'
import { useSeasonEvents, useSeasonPlan } from '../hooks/useSeasonPlan'

const plan: SeasonPlan = {
  as_of: '2026-10-06', event: 'christmas', year: 2026, safety_days: 7, suppliers: [],
  window: { start: '2026-11-13', end: '2026-12-27' }, last_year_window: { start: '2025-11-13', end: '2025-12-27' },
}

beforeEach(() => vi.clearAllMocks())

describe('useSeasonPlan', () => {
  it('loads the plan for an event', async () => {
    vi.mocked(api.getSeasonPlan).mockResolvedValue(plan)
    const query = { event: 'christmas' as const, safety_days: 7, all_products: false }

    const { result } = renderHook(() => useSeasonPlan('tok', query))

    await waitFor(() => expect(result.current.data).toEqual(plan))
    expect(api.getSeasonPlan).toHaveBeenCalledWith('tok', query)
  })

  it('waits for both ends of a custom range', () => {
    renderHook(() => useSeasonPlan('tok', { from: '2026-11-01', to: null, safety_days: 7, all_products: false }))
    expect(api.getSeasonPlan).not.toHaveBeenCalled()
  })

  it('surfaces errors', async () => {
    vi.mocked(api.getSeasonPlan).mockRejectedValue(new Error('this season is already over'))

    const { result } = renderHook(() => useSeasonPlan('tok', { event: 'easter', year: 2026, safety_days: 7, all_products: false }))

    await waitFor(() => expect(result.current.error).toBe('this season is already over'))
  })
})

describe('useSeasonEvents', () => {
  it('loads the next occurrence of every event', async () => {
    const events = [{ key: 'christmas' as const, year: 2026, date: '2026-12-25', window: { start: '2026-11-13', end: '2026-12-27' } }]
    vi.mocked(api.listSeasonEvents).mockResolvedValue(events)

    const { result } = renderHook(() => useSeasonEvents('tok'))

    await waitFor(() => expect(result.current.data).toEqual(events))
  })
})
