// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useSalesOrderLineChanges } from '../useSalesOrderLineChanges'
import { change } from '../../lineChanges/__tests__/fixtures'

vi.mock('../../api', async importOriginal => ({
  ...(await importOriginal<typeof import('../../api')>()),
  listSalesOrderLineChanges: vi.fn(),
}))
import { listSalesOrderLineChanges } from '../../api'

beforeEach(() => vi.clearAllMocks())

describe('useSalesOrderLineChanges', () => {
  it("loads the order's changes", async () => {
    vi.mocked(listSalesOrderLineChanges).mockResolvedValue([change()])

    const { result } = renderHook(() => useSalesOrderLineChanges('tok', 'so-1'))

    await waitFor(() => expect(result.current.changes).toEqual([change()]))
    expect(listSalesOrderLineChanges).toHaveBeenCalledWith('tok', 'so-1')
  })

  it('does nothing without an order', () => {
    renderHook(() => useSalesOrderLineChanges('tok', null))

    expect(listSalesOrderLineChanges).not.toHaveBeenCalled()
  })

  it('surfaces the API error', async () => {
    vi.mocked(listSalesOrderLineChanges).mockRejectedValue(new Error('sales order not found'))

    const { result } = renderHook(() => useSalesOrderLineChanges('tok', 'so-1'))

    await waitFor(() => expect(result.current.error).toBe('sales order not found'))
  })
})
