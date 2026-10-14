import { describe, it, expect } from 'vitest'
import { orderPackLines, packLineChanges, packLineState } from '../packLineOrder'

const lines = [
  { id: 'a', quantity: 2 },
  { id: 'b', quantity: 1 },
  { id: 'c', quantity: 3 },
  { id: 'd', quantity: 1 },
]
const ids = (ls: { id: string }[]) => ls.map(l => l.id)

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

  it('puts partly packed lines on top, then lines not packed, then done lines at the end', () => {
    const packed = new Map([['a', 2], ['c', 1], ['d', 1]])
    expect(ids(orderPackLines(lines, packed))).toEqual(['c', 'b', 'a', 'd'])
  })

  it('keeps the original order inside each group', () => {
    const packed = new Map([['b', 1], ['a', 1], ['c', 1]])
    // a and c partial (a before c), d todo, b done.
    expect(ids(orderPackLines(lines, packed))).toEqual(['a', 'c', 'd', 'b'])
  })

  it('leaves a settling line where it was before it completed', () => {
    const packed = new Map([['a', 2]])
    expect(ids(orderPackLines(lines, packed, new Map([['a', 'todo']])))).toEqual(['a', 'b', 'c', 'd'])
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
