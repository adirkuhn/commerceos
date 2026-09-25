// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useSessionCompletion } from '../useSessionCompletion'
import type { StockShortfall } from '../types'

interface Result { completed: boolean; stock_shortfalls?: StockShortfall[]; missing?: number }

const shortfall: StockShortfall = { scan_id: 'x1', sku_id: 'sku-9', name: 'Jam jar', needed: 2, short: 1 }

function setup(complete: (r?: Record<string, 'with_item' | 'without_item'>) => Promise<Result>) {
  const onAuthError = vi.fn()
  const setError = vi.fn()
  const hook = renderHook(() => useSessionCompletion(complete, onAuthError, setError))
  return { ...hook, onAuthError, setError }
}

describe('useSessionCompletion', () => {
  it('a completed result closes the review and is kept', async () => {
    const { result } = setup(() => Promise.resolve({ completed: true, missing: 0 }))
    act(() => result.current.setConfirmingFinish(true))

    await act(async () => { await result.current.confirmComplete() })

    expect(result.current.result).toEqual({ completed: true, missing: 0 })
    expect(result.current.confirmingFinish).toBe(false)
    expect(result.current.finishing).toBe(false)
  })

  it('a stock shortfall asks for decisions, then resubmits with them', async () => {
    const complete = vi.fn()
      .mockResolvedValueOnce({ completed: false, stock_shortfalls: [shortfall] })
      .mockResolvedValueOnce({ completed: true })
    const { result } = setup(complete)
    act(() => result.current.setConfirmingFinish(true))

    await act(async () => { await result.current.confirmComplete() })
    expect(result.current.stockShortfalls).toEqual([shortfall])
    expect(result.current.result).toBeNull()
    expect(result.current.confirmingFinish).toBe(true)

    await act(async () => { await result.current.confirmComplete({ x1: 'without_item' }) })
    expect(complete).toHaveBeenLastCalledWith({ x1: 'without_item' })
    expect(result.current.stockShortfalls).toBeNull()
    expect(result.current.result).toEqual({ completed: true })
  })

  it('a not-completed result without shortfalls still opens an empty list', async () => {
    const { result } = setup(() => Promise.resolve({ completed: false }))
    await act(async () => { await result.current.confirmComplete() })
    expect(result.current.stockShortfalls).toEqual([])
  })

  it('failures are reported, auth failures handed off', async () => {
    const failing = setup(() => Promise.reject(new Error('Server down')))
    await act(async () => { await failing.result.current.confirmComplete() })
    expect(failing.setError).toHaveBeenCalledWith('Server down')

    const fallback = setup(() => Promise.reject('x'))
    await act(async () => { await fallback.result.current.confirmComplete() })
    expect(fallback.setError).toHaveBeenCalledWith('Failed to complete')

    const auth = setup(() => Promise.reject(new Error('401')))
    await act(async () => { await auth.result.current.confirmComplete() })
    expect(auth.onAuthError).toHaveBeenCalled()
    expect(auth.setError).not.toHaveBeenCalled()
  })
})
