// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type React from 'react'

vi.mock('../../api', () => ({ deleteAllImportedProducts: vi.fn() }))

import * as api from '../../api'
import { useDeleteImportedProducts } from '../useDeleteImportedProducts'
import type { Connection } from '../../types'

const event = () => ({ preventDefault: vi.fn() }) as unknown as React.FormEvent
const woo: Connection = { id: 'c1', provider: 'woocommerce', store_url: '', status: 'connected', created_at: '' }

beforeEach(() => vi.clearAllMocks())

describe('useDeleteImportedProducts', () => {
  it('is confirmed only once DELETE is typed exactly', () => {
    const { result } = renderHook(() => useDeleteImportedProducts('tok', woo))
    expect(result.current.confirmed).toBe(false)
    act(() => result.current.setConfirmText('delete'))
    expect(result.current.confirmed).toBe(false)
    act(() => result.current.setConfirmText('DELETE'))
    expect(result.current.confirmed).toBe(true)
  })

  it('stores the result on success', async () => {
    const res = { deleted_sales_orders: 0, deleted_products: 3, deleted_categories: 1, errors: [] }
    vi.mocked(api.deleteAllImportedProducts).mockResolvedValue(res)
    const { result } = renderHook(() => useDeleteImportedProducts('tok', woo))
    await act(() => result.current.submit(event()))
    expect(api.deleteAllImportedProducts).toHaveBeenCalledWith('tok', 'c1')
    expect(result.current.result).toBe(res)
    expect(result.current.deleting).toBe(false)
  })

  it('surfaces a failure', async () => {
    vi.mocked(api.deleteAllImportedProducts).mockRejectedValue(new Error('db down'))
    const { result } = renderHook(() => useDeleteImportedProducts('tok', woo))
    await act(() => result.current.submit(event()))
    expect(result.current.error).toBe('db down')
    expect(result.current.result).toBeNull()
  })
})
