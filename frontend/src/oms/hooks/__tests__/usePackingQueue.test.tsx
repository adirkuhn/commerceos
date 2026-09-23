// @vitest-environment jsdom
import { renderHook, waitFor, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { usePackingQueue, PACKING_QUEUE_POLL_MS } from '../usePackingQueue'
import type { PackingQueue } from '../../types'

vi.mock('../../api', () => ({ listPackingQueue: vi.fn() }))
import * as api from '../../api'

const queue: PackingQueue = { entries: [{ sales_order: { id: 'so-1' } as PackingQueue['entries'][number]['sales_order'], lock: null }], total: 1 }

beforeEach(() => vi.clearAllMocks())
afterEach(() => vi.useRealTimers())

describe('usePackingQueue', () => {
  it('loads the first page of the queue', async () => {
    vi.mocked(api.listPackingQueue).mockResolvedValue(queue)

    const { result } = renderHook(() => usePackingQueue('tok'))

    await waitFor(() => expect(result.current.entries).toEqual(queue.entries))
    expect(api.listPackingQueue).toHaveBeenCalledWith('tok', { perPage: 100 })
    expect(result.current.loading).toBe(false)
  })

  it('surfaces the API error', async () => {
    vi.mocked(api.listPackingQueue).mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() => usePackingQueue('tok'))

    await waitFor(() => expect(result.current.error).toBe('boom'))
  })

  it('polls so locks stay current', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.mocked(api.listPackingQueue).mockResolvedValue(queue)

    renderHook(() => usePackingQueue('tok'))
    await waitFor(() => expect(api.listPackingQueue).toHaveBeenCalledTimes(1))
    await act(async () => { vi.advanceTimersByTime(PACKING_QUEUE_POLL_MS) })

    await waitFor(() => expect(api.listPackingQueue).toHaveBeenCalledTimes(2))
  })

  it('refresh reloads on demand', async () => {
    vi.mocked(api.listPackingQueue).mockResolvedValue(queue)
    const { result } = renderHook(() => usePackingQueue('tok'))
    await waitFor(() => expect(api.listPackingQueue).toHaveBeenCalledTimes(1))

    await act(async () => { await result.current.refresh() })

    expect(api.listPackingQueue).toHaveBeenCalledTimes(2)
  })
})
