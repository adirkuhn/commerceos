// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useOrderQuantities } from '../hooks/useOrderQuantities'
import type { RecommendationLine } from '../types'

const line = (sku_id: string, purchase_quantity: number) => ({ sku_id, purchase_quantity }) as RecommendationLine

describe('useOrderQuantities', () => {
  it('starts from the plan and orders only positive whole quantities', () => {
    const lines = [line('a', 5), line('b', 3), line('c', 2)]
    const { result } = renderHook(() => useOrderQuantities(lines))
    expect(result.current.orderLines).toEqual([{ sku_id: 'a', quantity: 5 }, { sku_id: 'b', quantity: 3 }, { sku_id: 'c', quantity: 2 }])

    act(() => {
      result.current.setTyped('a', '')
      result.current.setTyped('b', '1.5')
      result.current.setTyped('c', '7')
    })

    expect(result.current.typed('a')).toBe('')
    expect(result.current.quantity('b')).toBe(0)
    expect(result.current.orderLines).toEqual([{ sku_id: 'c', quantity: 7 }])
  })

  it('resets when the plan reloads', () => {
    const { result, rerender } = renderHook(({ lines }) => useOrderQuantities(lines), { initialProps: { lines: [line('a', 5)] } })
    act(() => result.current.setTyped('a', '1'))

    rerender({ lines: [line('a', 9)] })

    expect(result.current.quantity('a')).toBe(9)
  })
})
