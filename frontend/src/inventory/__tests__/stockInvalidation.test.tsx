// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { StockInvalidationProvider, useInvalidateStock, useOnStockInvalidated } from '../stockInvalidation'

const wrapper = ({ children }: { children: ReactNode }) => <StockInvalidationProvider>{children}</StockInvalidationProvider>

function useBoth(skuIds: string[], onInvalidate: () => void) {
  useOnStockInvalidated(skuIds, onInvalidate)
  return useInvalidateStock()
}

describe('stock invalidation', () => {
  it('notifies a subscriber when one of its SKUs is invalidated', () => {
    const onInvalidate = vi.fn()
    const { result } = renderHook(() => useBoth(['sku-a', 'sku-b'], onInvalidate), { wrapper })

    act(() => result.current(['sku-b', 'sku-z']))

    expect(onInvalidate).toHaveBeenCalledTimes(1)
  })

  it('does not notify a subscriber for unrelated SKUs', () => {
    const onInvalidate = vi.fn()
    const { result } = renderHook(() => useBoth(['sku-a'], onInvalidate), { wrapper })

    act(() => result.current(['sku-z']))

    expect(onInvalidate).not.toHaveBeenCalled()
  })

  it('always calls the latest callback and stops after unmount', () => {
    const first = vi.fn()
    const second = vi.fn()
    const { result, rerender, unmount } = renderHook(({ cb }) => useBoth(['sku-a'], cb), { wrapper, initialProps: { cb: first } })
    const invalidate = result.current

    rerender({ cb: second })
    act(() => invalidate(['sku-a']))
    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledTimes(1)

    unmount()
    act(() => invalidate(['sku-a']))
    expect(second).toHaveBeenCalledTimes(1)
  })

  it('follows a changed SKU list', () => {
    const onInvalidate = vi.fn()
    const { result, rerender } = renderHook(({ ids }) => useBoth(ids, onInvalidate), { wrapper, initialProps: { ids: ['sku-a'] } })

    rerender({ ids: ['sku-b'] })
    act(() => result.current(['sku-a']))
    expect(onInvalidate).not.toHaveBeenCalled()
    act(() => result.current(['sku-b']))
    expect(onInvalidate).toHaveBeenCalledTimes(1)
  })

  it('is a harmless no-op outside a provider', () => {
    const onInvalidate = vi.fn()
    const { result } = renderHook(() => useBoth(['sku-a'], onInvalidate))

    act(() => result.current(['sku-a']))

    expect(onInvalidate).not.toHaveBeenCalled()
  })
})
