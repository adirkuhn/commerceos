// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePickList } from '../usePickList'
import type { PickList } from '../../types'

vi.mock('../../api', () => ({ getPickList: vi.fn() }))
import * as api from '../../api'

const pickList: PickList = {
  items: [{ sku_id: 'sku-a', sku_code: 'A', barcode: '111', name: 'Apple', total_quantity: 2, orders: [] }],
}

beforeEach(() => vi.clearAllMocks())

describe('usePickList', () => {
  it('loads the pick list for the given orders', async () => {
    vi.mocked(api.getPickList).mockResolvedValue(pickList)

    const { result } = renderHook(() => usePickList('tok', ['so-1', 'so-2']))

    await waitFor(() => expect(result.current.pickList).toEqual(pickList))
    expect(api.getPickList).toHaveBeenCalledWith('tok', ['so-1', 'so-2'])
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  it('surfaces the API error message', async () => {
    vi.mocked(api.getPickList).mockRejectedValue(new Error('sales order not found'))

    const { result } = renderHook(() => usePickList('tok', ['missing']))

    await waitFor(() => expect(result.current.error).toBe('sales order not found'))
    expect(result.current.pickList).toBeNull()
  })

  it('does not call the API with no orders selected', () => {
    const { result } = renderHook(() => usePickList('tok', []))

    expect(api.getPickList).not.toHaveBeenCalled()
    expect(result.current.pickList).toBeNull()
  })

  it('does not refetch when rerendered with an equal id list', async () => {
    vi.mocked(api.getPickList).mockResolvedValue(pickList)

    const { result, rerender } = renderHook(({ ids }) => usePickList('tok', ids), { initialProps: { ids: ['so-1'] } })
    await waitFor(() => expect(result.current.pickList).toEqual(pickList))
    rerender({ ids: ['so-1'] })

    expect(api.getPickList).toHaveBeenCalledTimes(1)
  })
})
