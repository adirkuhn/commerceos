import { describe, it, expect } from 'vitest'
import { parseDays, COVERAGE, SAFETY } from '../planningWindow'

describe('parseDays', () => {
  it('reads a valid day count', () => {
    expect(parseDays('60', COVERAGE)).toBe(60)
    expect(parseDays('0', SAFETY)).toBe(0)
  })

  it('falls back to the default for a missing, invalid or out-of-range value', () => {
    expect(parseDays(null, COVERAGE)).toBe(30)
    expect(parseDays('abc', COVERAGE)).toBe(30)
    expect(parseDays('1.5', COVERAGE)).toBe(30)
    expect(parseDays('0', COVERAGE)).toBe(30)
    expect(parseDays('366', COVERAGE)).toBe(30)
    expect(parseDays('91', SAFETY)).toBe(7)
  })
})
