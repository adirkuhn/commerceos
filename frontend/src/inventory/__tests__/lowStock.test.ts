import { describe, it, expect } from 'vitest'
import { effectiveLowStockThreshold, stockTone } from '../lowStock'

describe('effectiveLowStockThreshold', () => {
  it("uses the product's own line, else the shop's, else the default", () => {
    expect(effectiveLowStockThreshold(1000, 5)).toBe(1000)
    expect(effectiveLowStockThreshold(0, 5)).toBe(0)
    expect(effectiveLowStockThreshold(null, 5)).toBe(5)
    expect(effectiveLowStockThreshold(undefined, undefined)).toBe(10)
  })
})

describe('stockTone', () => {
  it('is out at or below 0, low above 0 up to the line, else ok', () => {
    expect(stockTone(0, 10)).toBe('out')
    expect(stockTone(-2, 10)).toBe('out')
    expect(stockTone(1, 10)).toBe('low')
    expect(stockTone(10, 10)).toBe('low')
    expect(stockTone(11, 10)).toBe('ok')
  })

  it('a line of 0 never flags low', () => {
    expect(stockTone(1, 0)).toBe('ok')
  })
})
