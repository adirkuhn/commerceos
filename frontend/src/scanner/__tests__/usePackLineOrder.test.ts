// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, it, expect, vi } from 'vitest'
import { usePackLineOrder, PACK_LINE_SETTLE_MS } from '../usePackLineOrder'

const lines = [
  { id: 'a', quantity: 1 },
  { id: 'b', quantity: 2 },
  { id: 'c', quantity: 1 },
]
type Props = { packed: Map<string, number>; session: string }
const ids = (ls: { id: string }[]) => ls.map(l => l.id)

function setup(initial: Map<string, number>) {
  return renderHook(({ packed, session }: Props) => usePackLineOrder(session, lines, packed), {
    initialProps: { packed: initial, session: 's1' },
  })
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('usePackLineOrder', () => {
  it('opens a fresh order as it is, without highlights', () => {
    const { result } = setup(new Map())
    expect(ids(result.current.lines)).toEqual(['a', 'b', 'c'])
    expect(result.current.highlight.size).toBe(0)
  })

  it('opens a resumed order already sorted, without highlights', () => {
    const { result } = setup(new Map([['a', 1], ['b', 1]]))
    expect(ids(result.current.lines)).toEqual(['b', 'c', 'a'])
    expect(result.current.highlight.size).toBe(0)
  })

  it('flashes a completed line in place, then moves it to the end', () => {
    const { result, rerender } = setup(new Map())

    rerender({ packed: new Map([['a', 1]]), session: 's1' })
    expect(ids(result.current.lines)).toEqual(['a', 'b', 'c'])
    expect(result.current.highlight.get('a')).toBe('done')

    act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })
    expect(ids(result.current.lines)).toEqual(['b', 'c', 'a'])
    expect(result.current.highlight.has('a')).toBe(false)
  })

  it('moves a partly packed line to the top right away, with a pulse', () => {
    const { result, rerender } = setup(new Map())

    rerender({ packed: new Map([['b', 1]]), session: 's1' })
    expect(ids(result.current.lines)).toEqual(['b', 'a', 'c'])
    expect(result.current.highlight.get('b')).toBe('partial')

    act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })
    expect(result.current.highlight.size).toBe(0)
  })

  it('does not flash lines when another order is opened', () => {
    const { result, rerender } = setup(new Map())

    rerender({ packed: new Map([['a', 1]]), session: 's2' })
    expect(ids(result.current.lines)).toEqual(['b', 'c', 'a'])
    expect(result.current.highlight.size).toBe(0)
  })
})
