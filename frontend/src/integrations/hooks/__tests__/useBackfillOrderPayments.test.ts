// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../api', () => ({
  backfillWooOrderPayments: vi.fn(),
}))

import * as api from '../../api'
import { useBackfillOrderPayments } from '../useBackfillOrderPayments'

beforeEach(() => vi.clearAllMocks())

describe('useBackfillOrderPayments', () => {
  it('runs batch after batch until there is no next cursor, adding up the totals', async () => {
    vi.mocked(api.backfillWooOrderPayments)
      .mockResolvedValueOnce({ checked: 25, updated: 20, errors: ['woo order 9: fetch: boom'], next_cursor: 'c1' })
      .mockResolvedValueOnce({ checked: 7, updated: 5, errors: [], next_cursor: null })
    const { result } = renderHook(() => useBackfillOrderPayments('tok', 'conn-1'))

    await act(async () => { await result.current.run() })

    expect(api.backfillWooOrderPayments).toHaveBeenNthCalledWith(1, 'tok', 'conn-1', null)
    expect(api.backfillWooOrderPayments).toHaveBeenNthCalledWith(2, 'tok', 'conn-1', 'c1')
    expect(result.current.progress).toEqual({ checked: 32, updated: 25, errors: ['woo order 9: fetch: boom'] })
    expect(result.current.running).toBe(false)
    expect(result.current.done).toBe(true)
    expect(result.current.error).toBe('')
  })

  it('shows progress while batches are still running', async () => {
    let releaseSecond: (v: { checked: number; updated: number; errors: string[]; next_cursor: string | null }) => void = () => {}
    vi.mocked(api.backfillWooOrderPayments)
      .mockResolvedValueOnce({ checked: 25, updated: 3, errors: [], next_cursor: 'c1' })
      .mockReturnValueOnce(new Promise(resolve => { releaseSecond = resolve }))
    const { result } = renderHook(() => useBackfillOrderPayments('tok', 'conn-1'))

    let running: Promise<void> = Promise.resolve()
    act(() => { running = result.current.run() })
    await waitFor(() => expect(result.current.progress.checked).toBe(25))
    expect(result.current.running).toBe(true)

    await act(async () => { releaseSecond({ checked: 1, updated: 1, errors: [], next_cursor: null }); await running })
    expect(result.current.progress.checked).toBe(26)
  })

  it('stops and reports a failed request, keeping what was done so far', async () => {
    vi.mocked(api.backfillWooOrderPayments)
      .mockResolvedValueOnce({ checked: 25, updated: 10, errors: [], next_cursor: 'c1' })
      .mockRejectedValueOnce(new Error('woocommerce request failed'))
    const { result } = renderHook(() => useBackfillOrderPayments('tok', 'conn-1'))

    await act(async () => { await result.current.run() })

    expect(result.current.error).toBe('woocommerce request failed')
    expect(result.current.progress.updated).toBe(10)
    expect(result.current.running).toBe(false)
    expect(result.current.done).toBe(false)
  })
})
