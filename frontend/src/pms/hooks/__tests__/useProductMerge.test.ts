// @vitest-environment jsdom
import { renderHook, waitFor, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useProductMerge } from '../useProductMerge'
import type { ProductMergePreview } from '../../types'

vi.mock('../../api', () => ({ previewProductMerge: vi.fn(), mergeProduct: vi.fn() }))
import * as api from '../../api'

const preview = (over: Partial<ProductMergePreview> = {}): ProductMergePreview => ({
  source: { id: 'dup', name: 'Dup', skus: [{ id: 'd1', sku_code: 'D-1', barcode: '', on_hand: 4 }, { id: 'd2', sku_code: 'D-2', barcode: '', on_hand: 1 }] },
  target: { id: 'orig', name: 'Orig', skus: [{ id: 'o1', sku_code: 'O-1', barcode: '', on_hand: 6 }, { id: 'o2', sku_code: 'O-2', barcode: '', on_hand: 0 }] },
  sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }],
  targets: [],
  ...over,
})

beforeEach(() => vi.clearAllMocks())

describe('useProductMerge', () => {
  it('does nothing until a target is chosen', () => {
    const { result } = renderHook(() => useProductMerge('t', 'dup', null))
    expect(result.current.preview).toBeNull()
    expect(api.previewProductMerge).not.toHaveBeenCalled()
  })

  it('loads the suggested matching for a target', async () => {
    vi.mocked(api.previewProductMerge).mockResolvedValue(preview())
    const { result } = renderHook(() => useProductMerge('t', 'dup', 'orig'))

    await waitFor(() => expect(result.current.preview).not.toBeNull())
    expect(api.previewProductMerge).toHaveBeenCalledWith('t', 'dup', { target_product_id: 'orig' })
    expect(result.current.mapping).toEqual({ d1: 'o1' })
    expect(result.current.complete).toBe(false)
  })

  it('re-previews with the full mapping once every SKU is matched', async () => {
    vi.mocked(api.previewProductMerge).mockResolvedValue(preview())
    const { result } = renderHook(() => useProductMerge('t', 'dup', 'orig'))
    await waitFor(() => expect(result.current.preview).not.toBeNull())

    act(() => result.current.match('d2', 'o2'))

    expect(result.current.complete).toBe(true)
    await waitFor(() => expect(api.previewProductMerge).toHaveBeenLastCalledWith('t', 'dup', {
      target_product_id: 'orig',
      sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }, { source_sku_id: 'd2', target_sku_id: 'o2' }],
    }))
  })

  it('merges with the mapping and surfaces a failure', async () => {
    vi.mocked(api.previewProductMerge).mockResolvedValue(preview({ sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }, { source_sku_id: 'd2', target_sku_id: 'o1' }] }))
    vi.mocked(api.mergeProduct).mockResolvedValue({ target_product_id: 'orig', skus: [] })
    const { result } = renderHook(() => useProductMerge('t', 'dup', 'orig'))
    await waitFor(() => expect(result.current.complete).toBe(true))

    await act(async () => { await result.current.merge() })
    expect(api.mergeProduct).toHaveBeenCalledWith('t', 'dup', {
      target_product_id: 'orig',
      sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }, { source_sku_id: 'd2', target_sku_id: 'o1' }],
    })

    vi.mocked(api.mergeProduct).mockRejectedValue(new Error('only an owner or admin can merge products'))
    await act(async () => { await expect(result.current.merge()).rejects.toThrow('only an owner') })
  })

  it('shows a preview failure as an error', async () => {
    vi.mocked(api.previewProductMerge).mockRejectedValue(new Error('product not found'))
    const { result } = renderHook(() => useProductMerge('t', 'dup', 'orig'))
    await waitFor(() => expect(result.current.error).toBe('product not found'))
  })
})
