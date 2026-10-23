// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import type { ReactNode } from 'react'
import { act, renderHook } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { useQuoteLines } from '../hooks/useQuoteLines'

const wrapper = (initial: string) => ({ children }: { children: ReactNode }) =>
  <MemoryRouter initialEntries={[initial]}>{children}</MemoryRouter>

function render(initial = '/price-list?tab=quote') {
  return renderHook(() => ({ quote: useQuoteLines(), location: useLocation() }), { wrapper: wrapper(initial) })
}

describe('useQuoteLines', () => {
  it('reads the lines from the URL', () => {
    const { result } = render('/price-list?lines=p1.s1.5.899')
    expect(result.current.quote.lines).toEqual([{ productId: 'p1', skuId: 's1', quantity: 5, unitPrice: 899 }])
  })

  it('adds a SKU once with quantity 1, keeping other params', () => {
    const { result } = render()
    act(() => result.current.quote.add('p1', 's1'))
    act(() => result.current.quote.add('p1', 's1'))
    expect(result.current.quote.lines).toEqual([{ productId: 'p1', skuId: 's1', quantity: 1 }])
    expect(result.current.location.search).toContain('tab=quote')
  })

  it('sets quantity and a typed price, and clears the price back to the default', () => {
    const { result } = render('/price-list?lines=p1.s1.1')
    act(() => result.current.quote.setQuantity('s1', 250))
    act(() => result.current.quote.setUnitPrice('s1', 799))
    expect(result.current.quote.lines[0]).toEqual({ productId: 'p1', skuId: 's1', quantity: 250, unitPrice: 799 })
    act(() => result.current.quote.setUnitPrice('s1', undefined))
    expect(result.current.quote.lines[0]).toEqual({ productId: 'p1', skuId: 's1', quantity: 250 })
  })

  it('removes a line and clears the param when the quote is empty', () => {
    const { result } = render('/price-list?lines=p1.s1.1,p2.s2.3')
    act(() => result.current.quote.remove('s1'))
    expect(result.current.quote.lines.map(l => l.skuId)).toEqual(['s2'])
    act(() => result.current.quote.clear())
    expect(result.current.quote.lines).toEqual([])
    expect(result.current.location.search).not.toContain('lines')
  })
})
