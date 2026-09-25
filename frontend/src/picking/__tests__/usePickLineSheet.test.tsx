// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { usePickLineSheet } from '../usePickLineSheet'
import type { PickingSessionLineItem } from '../../scanner/types'

const line: PickingSessionLineItem = { id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'W', barcode: '', quantity: 5, price_minor_units: 0, total_minor_units: 0 }

describe('usePickLineSheet', () => {
  it('opens prefilled with the quantity still to pick', () => {
    const { result } = renderHook(() => usePickLineSheet())
    act(() => result.current.open(line, 2))
    expect(result.current.pickLine).toEqual({ lineItem: line, pickedSoFar: 2 })
    expect(result.current.pickQty).toBe(3)
  })

  it('prefills at least 1 even when the line is already covered', () => {
    const { result } = renderHook(() => usePickLineSheet())
    act(() => result.current.open(line, 5))
    expect(result.current.pickQty).toBe(1)
  })

  it('does not open for an unlinked line with no SKU', () => {
    const { result } = renderHook(() => usePickLineSheet())
    act(() => result.current.open({ ...line, sku_id: null }, 0))
    expect(result.current.pickLine).toBeNull()
  })

  it('closes', () => {
    const { result } = renderHook(() => usePickLineSheet())
    act(() => result.current.open(line, 0))
    act(() => result.current.close())
    expect(result.current.pickLine).toBeNull()
  })
})
