import { describe, it, expect } from 'vitest'
import { moveItem, removeItem, replaceItem } from '../listOps'

describe('list ops', () => {
  it('moves, replaces and removes without mutating', () => {
    const list = ['a', 'b', 'c'] as const
    expect(moveItem(list, 0, 1)).toEqual(['b', 'a', 'c'])
    expect(moveItem(list, 0, -1)).toEqual(['a', 'b', 'c'])
    expect(moveItem(list, 2, 1)).toEqual(['a', 'b', 'c'])
    expect(replaceItem(list, 1, 'x')).toEqual(['a', 'x', 'c'])
    expect(removeItem(list, 0)).toEqual(['b', 'c'])
    expect(list).toEqual(['a', 'b', 'c'])
  })
})
