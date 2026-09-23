import { describe, it, expect } from 'vitest'
import { createGoToSequence, GO_TO_TIMEOUT_MS } from '../goToSequence'

const TARGETS = { d: 'dashboard', p: 'products' } as const

describe('createGoToSequence', () => {
  it('matches g followed by a target letter', () => {
    const seq = createGoToSequence(TARGETS)
    expect(seq.press('g', 0)).toBeNull()
    expect(seq.press('p', 100)).toBe('products')
  })

  it('ignores a target letter without a leading g', () => {
    const seq = createGoToSequence(TARGETS)
    expect(seq.press('p', 0)).toBeNull()
  })

  it('forgets the g after the timeout', () => {
    const seq = createGoToSequence(TARGETS)
    seq.press('g', 0)
    expect(seq.press('p', GO_TO_TIMEOUT_MS + 1)).toBeNull()
  })

  it('resets after an unknown second key', () => {
    const seq = createGoToSequence(TARGETS)
    seq.press('g', 0)
    expect(seq.press('x', 10)).toBeNull()
    expect(seq.press('p', 20)).toBeNull()
  })

  it('is case-insensitive', () => {
    const seq = createGoToSequence(TARGETS)
    seq.press('G', 0)
    expect(seq.press('D', 10)).toBe('dashboard')
  })

  it('needs a fresh g for each jump', () => {
    const seq = createGoToSequence(TARGETS)
    seq.press('g', 0)
    seq.press('d', 10)
    expect(seq.press('p', 20)).toBeNull()
  })
})
