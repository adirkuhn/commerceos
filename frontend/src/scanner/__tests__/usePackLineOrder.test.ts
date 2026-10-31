// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, it, expect, vi } from 'vitest'
import { usePackLineOrder, PACK_LINE_SETTLE_MS } from '../usePackLineOrder'

// c and a are finished in one scan; e (2) and b (3) are packed in steps.
const lines = [
  { id: 'a', quantity: 1 },
  { id: 'b', quantity: 3 },
  { id: 'c', quantity: 1 },
  { id: 'd', quantity: 1 },
  { id: 'e', quantity: 2 },
  { id: 'f', quantity: 1 },
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
    expect(ids(result.current.lines)).toEqual(['a', 'b', 'c', 'd', 'e', 'f'])
    expect(result.current.highlight.size).toBe(0)
  })

  it('opens a resumed order sorted (partly packed, then not packed, then done), without highlights', () => {
    const { result } = setup(new Map([['a', 1], ['b', 1]]))
    expect(ids(result.current.lines)).toEqual(['b', 'c', 'd', 'e', 'f', 'a'])
    expect(result.current.highlight.size).toBe(0)
  })

  // The sequence agreed for big orders (2026-10-31).
  it('keeps what was just scanned on top', () => {
    const { result, rerender } = setup(new Map())
    const packed = new Map<string, number>()
    const scan = (id: string) => {
      packed.set(id, (packed.get(id) ?? 0) + 1)
      rerender({ packed: new Map(packed), session: 's1' })
    }
    const settle = () => act(() => { vi.advanceTimersByTime(PACK_LINE_SETTLE_MS) })

    // c finished in one scan: on top, green, and it stays green.
    scan('c')
    expect(ids(result.current.lines)).toEqual(['c', 'a', 'b', 'd', 'e', 'f'])
    settle()
    expect(result.current.highlight.get('c')).toBe('done')

    // e partly: e on top (amber), c goes to the end.
    scan('e')
    expect(ids(result.current.lines)).toEqual(['e', 'a', 'b', 'd', 'f', 'c'])
    expect(result.current.highlight.get('e')).toBe('partial')
    expect(result.current.highlight.has('c')).toBe(false)
    settle()

    // a finished in one scan: on top, then the partly packed e.
    scan('a')
    expect(ids(result.current.lines)).toEqual(['a', 'e', 'b', 'd', 'f', 'c'])
    expect(result.current.highlight.get('a')).toBe('done')
    settle()

    // b 1/3: partly packed lines in first-scanned order (e, b); done lines
    // at the end in the order's own order (a, c).
    scan('b')
    expect(ids(result.current.lines)).toEqual(['e', 'b', 'd', 'f', 'a', 'c'])
    expect(result.current.highlight.has('a')).toBe(false)
    settle()

    // e finished from partly packed: flashes green where it is, then goes
    // straight to the end -- no top spot.
    scan('e')
    expect(ids(result.current.lines)).toEqual(['e', 'b', 'd', 'f', 'a', 'c'])
    expect(result.current.highlight.get('e')).toBe('done')
    settle()
    expect(ids(result.current.lines)).toEqual(['b', 'd', 'f', 'a', 'c', 'e'])
    expect(result.current.highlight.has('e')).toBe(false)

    // b 2/3: b stays first.
    scan('b')
    expect(ids(result.current.lines)).toEqual(['b', 'd', 'f', 'a', 'c', 'e'])
  })

  it('does not flash lines when another order is opened', () => {
    const { result, rerender } = setup(new Map())

    rerender({ packed: new Map([['a', 1]]), session: 's2' })
    expect(ids(result.current.lines)).toEqual(['b', 'c', 'd', 'e', 'f', 'a'])
    expect(result.current.highlight.size).toBe(0)
  })
})
