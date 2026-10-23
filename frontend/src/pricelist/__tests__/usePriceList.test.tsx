// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'

vi.mock('../api', () => ({ listPriceList: vi.fn() }))
import * as api from '../api'
import { usePriceList } from '../hooks/usePriceList'

const page = { rows: [], total: 0, pricesIncludeTax: true }

beforeEach(() => vi.clearAllMocks())

describe('usePriceList', () => {
  it('loads the page for the query', async () => {
    vi.mocked(api.listPriceList).mockResolvedValue(page)
    const query = { page: 2, perPage: 50, search: 'fei' }

    const { result } = renderHook(() => usePriceList('tok', query))

    await waitFor(() => expect(result.current.data).toEqual(page))
    expect(api.listPriceList).toHaveBeenCalledWith('tok', query)
  })

  it('surfaces errors', async () => {
    vi.mocked(api.listPriceList).mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() => usePriceList('tok', {}))

    await waitFor(() => expect(result.current.error).toBe('boom'))
  })

  it('does nothing while disabled', () => {
    renderHook(() => usePriceList('tok', {}, false))
    expect(api.listPriceList).not.toHaveBeenCalled()
  })
})
