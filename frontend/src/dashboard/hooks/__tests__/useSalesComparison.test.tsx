// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { SalesComparison, SalesPeriod } from '../../types'

vi.mock('../../api', () => ({ getSalesComparison: vi.fn() }))

import * as api from '../../api'
import { useSalesComparison } from '../useSalesComparison'

function comparison(period: SalesPeriod): SalesComparison {
  const figures = { from: '', to: '', revenue_minor_units: 0, order_count: 0, by_channel: [] }
  return { period, currency_code: 'EUR', change_percent: null, current: figures, previous: figures }
}

beforeEach(() => { vi.clearAllMocks() })

describe('useSalesComparison', () => {
  it('loads the comparison for the period', async () => {
    vi.mocked(api.getSalesComparison).mockResolvedValue(comparison('today'))
    const { result } = renderHook(() => useSalesComparison('tok', 'today'))

    await waitFor(() => expect(result.current.data?.period).toBe('today'))
    expect(api.getSalesComparison).toHaveBeenCalledWith('tok', 'today')
    expect(result.current.error).toBeNull()
  })

  it('reloads when the period changes, keeping only the latest answer', async () => {
    let resolveSlow: (c: SalesComparison) => void = () => {}
    vi.mocked(api.getSalesComparison)
      .mockImplementationOnce(() => new Promise(r => { resolveSlow = r }))
      .mockResolvedValueOnce(comparison('this_week'))
    const { result, rerender } = renderHook(({ p }) => useSalesComparison('tok', p), { initialProps: { p: 'today' as SalesPeriod } })

    rerender({ p: 'this_week' })
    await waitFor(() => expect(result.current.data?.period).toBe('this_week'))
    resolveSlow(comparison('today'))
    await new Promise(r => setTimeout(r, 0))

    expect(result.current.data?.period).toBe('this_week')
  })

  it('surfaces an API error', async () => {
    vi.mocked(api.getSalesComparison).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useSalesComparison('tok', 'this_month'))

    await waitFor(() => expect(result.current.error).toBeTruthy())
    expect(result.current.data).toBeNull()
  })

  it('does nothing without a token', () => {
    renderHook(() => useSalesComparison('', 'today'))
    expect(api.getSalesComparison).not.toHaveBeenCalled()
  })
})
