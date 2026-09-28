import { describe, it, expect, vi } from 'vitest'
import { ValueStore } from '../valueStore'

describe('ValueStore', () => {
  it('notifies subscribers of changes only', () => {
    const s = new ValueStore(1)
    const fn = vi.fn()
    const off = s.subscribe(fn)
    s.set(1)
    expect(fn).not.toHaveBeenCalled()
    s.set(2)
    expect(s.snapshot()).toBe(2)
    expect(fn).toHaveBeenCalledTimes(1)
    off()
    s.set(3)
    expect(fn).toHaveBeenCalledTimes(1)
  })
})
