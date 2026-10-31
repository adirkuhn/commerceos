import { describe, it, expect } from 'vitest'
import { orderPackLines, packLineChanges, packLineState, type PackLineHistory } from '../packLineOrder'

const lines = [
  { id: 'a', quantity: 2 },
  { id: 'b', quantity: 1 },
  { id: 'c', quantity: 3 },
  { id: 'd', quantity: 1 },
]
const ids = (ls: { id: string }[]) => ls.map(l => l.id)
const history = (h: Partial<PackLineHistory>): PackLineHistory => ({ partialOrder: [], settling: new Map(), ...h })

describe('packLineState', () => {
  it('is todo, partial or done by packed quantity', () => {
    expect(packLineState({ id: 'a', quantity: 2 }, 0)).toBe('todo')
    expect(packLineState({ id: 'a', quantity: 2 }, 1)).toBe('partial')
    expect(packLineState({ id: 'a', quantity: 2 }, 2)).toBe('done')
    expect(packLineState({ id: 'a', quantity: 2 }, 3)).toBe('done')
  })
})

describe('orderPackLines', () => {
  it('keeps the order as it is when nothing is packed', () => {
    expect(ids(orderPackLines(lines, new Map()))).toEqual(['a', 'b', 'c', 'd'])
  })

  it('puts partly packed lines first, then lines not packed, then done lines in the order\'s own order', () => {
    const packed = new Map([['d', 1], ['c', 1], ['b', 1]])
    expect(ids(orderPackLines(lines, packed))).toEqual(['c', 'a', 'b', 'd'])
  })

  it('puts the line just finished in one scan on top, before the partly packed ones', () => {
    const packed = new Map([['c', 1], ['b', 1]])
    expect(ids(orderPackLines(lines, packed, history({ top: 'b' })))).toEqual(['b', 'c', 'a', 'd'])
  })

  it('orders partly packed lines by when they were first scanned', () => {
    const packed = new Map([['a', 1], ['c', 1]])
    expect(ids(orderPackLines(lines, packed, history({ partialOrder: ['c', 'a'] })))).toEqual(['c', 'a', 'b', 'd'])
  })

  it('ignores a top line that is no longer done (a scan was removed)', () => {
    expect(ids(orderPackLines(lines, new Map(), history({ top: 'b' })))).toEqual(['a', 'b', 'c', 'd'])
  })

  it('leaves a settling line where it was before it completed', () => {
    const packed = new Map([['a', 2], ['c', 1]])
    const h = history({ partialOrder: ['a', 'c'], settling: new Map([['a', 'partial']]) })
    expect(ids(orderPackLines(lines, packed, h))).toEqual(['a', 'c', 'b', 'd'])
  })
})

describe('packLineChanges', () => {
  it('reports lines that just completed and lines that just got partly packed', () => {
    const before = new Map([['c', 1]])
    const after = new Map([['a', 1], ['b', 1], ['c', 2]])
    expect(packLineChanges(lines, before, after)).toEqual({
      completed: new Map([['b', 'todo']]),
      advanced: ['a', 'c'],
    })
  })

  it('reports nothing when a scan is removed', () => {
    expect(packLineChanges(lines, new Map([['b', 1]]), new Map())).toEqual({ completed: new Map(), advanced: [] })
  })
})
