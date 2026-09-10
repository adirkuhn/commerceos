import { describe, expect, it } from 'vitest'
import { formatMoney, slugify, levenshtein, highlightMatch, cn } from '../utils'

describe('formatMoney', () => {
  it('formats integer minor units as currency', () => {
    expect(formatMoney(24900, 'USD')).toBe('$249.00')
  })

  it('respects the given currency', () => {
    expect(formatMoney(10000, 'EUR', 'en-US')).toBe('€100.00')
  })
})

describe('slugify', () => {
  it('lowercases and hyphenates', () => {
    expect(slugify('Summit Shell Jacket')).toBe('summit-shell-jacket')
  })

  it('strips non-alphanumeric characters', () => {
    expect(slugify("Rivet & Co.'s Best Bag!")).toBe('rivet-co-s-best-bag')
  })
})

describe('levenshtein', () => {
  it('returns 0 for identical strings', () => {
    expect(levenshtein('jacket', 'jacket')).toBe(0)
  })

  it('counts a single substitution as distance 1', () => {
    expect(levenshtein('jaket', 'jacket')).toBe(1)
  })

  it('handles empty strings', () => {
    expect(levenshtein('', 'abc')).toBe(3)
    expect(levenshtein('abc', '')).toBe(3)
  })
})

describe('highlightMatch', () => {
  it('splits text around a case-insensitive match', () => {
    expect(highlightMatch('Summit Shell Jacket', 'shell')).toEqual({ before: 'Summit ', match: 'Shell', after: ' Jacket' })
  })

  it('returns null when there is no match', () => {
    expect(highlightMatch('Summit Shell Jacket', 'boots')).toBeNull()
  })

  it('returns null for an empty query', () => {
    expect(highlightMatch('Summit Shell Jacket', '  ')).toBeNull()
  })
})

describe('cn', () => {
  it('merges tailwind classes, letting the later one win', () => {
    expect(cn('px-2 py-1', 'px-4')).toBe('py-1 px-4')
  })

  it('drops falsy values', () => {
    expect(cn('a', false && 'b', undefined, 'c')).toBe('a c')
  })
})
