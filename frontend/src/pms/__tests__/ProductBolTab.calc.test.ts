import { describe, it, expect } from 'vitest'
import { computeBolProfit } from '../ProductBolTab'

describe('computeBolProfit', () => {
  it('computes tax, commission, profit, and margin for known inputs', () => {
    // sell 19.99, cost 8.00, tax 21%, commission fixed 0.30 + 12.5%, shipping 3.50
    const result = computeBolProfit({
      sellPrice: 19.99, costPrice: 8.00, taxRatePercent: 21,
      commissionFixedAmount: 0.30, commissionPercent: 12.5, shippingCost: 3.50,
    })

    // tax = 19.99 * 0.21 = 4.1979
    expect(result.taxAmount).toBeCloseTo(4.1979, 4)
    // commission = 0.30 + 19.99 * 0.125 = 0.30 + 2.49875 = 2.79875
    expect(result.commissionAmount).toBeCloseTo(2.79875, 4)
    // profit = 19.99 - 8.00 - 4.1979 - 2.79875 - 3.50 = 1.49335
    expect(result.profit).toBeCloseTo(1.49335, 4)
    // margin = profit / sellPrice * 100 = 1.49335 / 19.99 * 100 ≈ 7.4705%
    expect(result.marginPercent).toBeCloseTo(7.4705, 3)
  })

  it('returns zero margin (not NaN/Infinity) when sell price is zero', () => {
    const result = computeBolProfit({
      sellPrice: 0, costPrice: 5, taxRatePercent: 21,
      commissionFixedAmount: 0, commissionPercent: 10, shippingCost: 2,
    })

    expect(result.marginPercent).toBe(0)
    expect(Number.isFinite(result.profit)).toBe(true)
  })

  it('a negative profit (selling at a loss) is reported as negative, not clamped to zero', () => {
    const result = computeBolProfit({
      sellPrice: 10, costPrice: 15, taxRatePercent: 0,
      commissionFixedAmount: 0, commissionPercent: 0, shippingCost: 0,
    })

    expect(result.profit).toBeCloseTo(-5, 4)
    expect(result.marginPercent).toBeCloseTo(-50, 4)
  })
})
