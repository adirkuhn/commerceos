// @vitest-environment jsdom
import { renderHook, waitFor, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useOrderDifferences } from '../useOrderDifferences'
import { useWooOrderAudit } from '../useWooOrderAudit'
import type { OrderDifference } from '../../orderDifferences'

vi.mock('../../orderDifferences', async importOriginal => ({
  ...(await importOriginal<typeof import('../../orderDifferences')>()),
  listOrderDifferences: vi.fn(),
  dismissOrderDifference: vi.fn(),
  auditWooOrderDifferences: vi.fn(),
}))
import { listOrderDifferences, dismissOrderDifference, auditWooOrderDifferences } from '../../orderDifferences'

const diff = { id: 'd-1', sales_order_id: 'so-1', order_number: '25922' } as OrderDifference

beforeEach(() => vi.clearAllMocks())

describe('useOrderDifferences', () => {
  it('loads a page', async () => {
    vi.mocked(listOrderDifferences).mockResolvedValue({ differences: [diff], total: 1 })

    const { result } = renderHook(() => useOrderDifferences('tok', 2))

    await waitFor(() => expect(result.current.differences).toEqual([diff]))
    expect(listOrderDifferences).toHaveBeenCalledWith('tok', 2, 20)
    expect(result.current.total).toBe(1)
  })

  it('surfaces the API error', async () => {
    vi.mocked(listOrderDifferences).mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() => useOrderDifferences('tok', 1))

    await waitFor(() => expect(result.current.error).toBe('boom'))
  })

  it('dismisses and reloads', async () => {
    vi.mocked(listOrderDifferences).mockResolvedValueOnce({ differences: [diff], total: 1 }).mockResolvedValueOnce({ differences: [], total: 0 })
    vi.mocked(dismissOrderDifference).mockResolvedValue(null)
    const { result } = renderHook(() => useOrderDifferences('tok', 1))
    await waitFor(() => expect(result.current.total).toBe(1))

    await act(async () => { await result.current.dismiss('d-1') })

    expect(dismissOrderDifference).toHaveBeenCalledWith('tok', 'd-1')
    expect(result.current.total).toBe(0)
  })
})

describe('useWooOrderAudit', () => {
  it('walks every batch until the cursor runs out, counting what was checked', async () => {
    vi.mocked(auditWooOrderDifferences)
      .mockResolvedValueOnce({ checked: 25, errors: [], next_cursor: 'c-2' })
      .mockResolvedValueOnce({ checked: 7, errors: ['woo order 999: fetch: not found'], next_cursor: '' })
    const onBatch = vi.fn()
    const { result } = renderHook(() => useWooOrderAudit('tok', onBatch))

    await act(async () => { await result.current.run('conn-1') })

    expect(auditWooOrderDifferences).toHaveBeenNthCalledWith(1, 'tok', 'conn-1', '')
    expect(auditWooOrderDifferences).toHaveBeenNthCalledWith(2, 'tok', 'conn-1', 'c-2')
    expect(result.current.checked).toBe(32)
    expect(result.current.failures).toEqual(['woo order 999: fetch: not found'])
    expect(result.current.running).toBe(false)
    expect(result.current.done).toBe(true)
    expect(onBatch).toHaveBeenCalledTimes(2)
  })

  it('stops and reports when a batch fails', async () => {
    vi.mocked(auditWooOrderDifferences).mockRejectedValue(new Error('connection not found'))
    const { result } = renderHook(() => useWooOrderAudit('tok', vi.fn()))

    await act(async () => { await result.current.run('conn-1') })

    expect(result.current.error).toBe('connection not found')
    expect(result.current.running).toBe(false)
    expect(result.current.done).toBe(false)
  })
})
