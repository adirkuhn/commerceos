import { describe, it, expect } from 'vitest'
import { liveMarginInsight } from '../liveMarginInsight'
import type { SKUMarginInsight } from '../types'

// Saved: price 10.00, sale 8.00, cost 7.00, 5 on hand.
const saved: SKUMarginInsight = {
  cost_price_available: true, cost_price_minor_units: 700, cost_price_source: 'computed',
  effective_target_markup_basis_points: 3000, recommended_price_minor_units: 910,
  profit_at_price_minor_units: 300, margin_at_price_basis_points: 3000, markup_at_price_basis_points: 4286,
  profit_at_sale_price_minor_units: 100, margin_at_sale_price_basis_points: 1250, markup_at_sale_price_basis_points: 1429,
  total_on_hand_quantity: 5, total_profit_potential_minor_units: 500, shrinkage_minor_units: 0,
}
const savedPrices = { savedPriceMinorUnits: 1000, savedSalePriceMinorUnits: 800 }

describe('liveMarginInsight', () => {
  it('without a draft: the saved figures, not unsaved', () => {
    expect(liveMarginInsight(saved, undefined)).toEqual({ insight: saved, unsaved: false })
  })

  it('a draft equal to the saved prices is not unsaved', () => {
    const live = liveMarginInsight(saved, { price: '10.00', salePrice: '8', ...savedPrices })
    expect(live.unsaved).toBe(false)
    expect(live.insight).toEqual(saved)
  })

  it('a typed price recomputes profit, margin and markup at it', () => {
    const live = liveMarginInsight(saved, { price: '14', salePrice: '8.00', ...savedPrices })
    expect(live.unsaved).toBe(true)
    expect(live.insight.profit_at_price_minor_units).toBe(700)
    expect(live.insight.margin_at_price_basis_points).toBe(5000)
    expect(live.insight.markup_at_price_basis_points).toBe(10000)
    expect(live.insight.profit_at_sale_price_minor_units).toBe(100)
    expect(live.insight.total_profit_potential_minor_units).toBe(500) // the sale price still sells
  })

  it('a typed sale price recomputes the sale row and the profit potential', () => {
    const live = liveMarginInsight(saved, { price: '10', salePrice: '6.00', ...savedPrices })
    expect(live.insight.profit_at_sale_price_minor_units).toBe(-100)
    expect(live.insight.margin_at_sale_price_basis_points).toBe(-1667)
    expect(live.insight.markup_at_sale_price_basis_points).toBe(-1429)
    expect(live.insight.total_profit_potential_minor_units).toBe(-500)
  })

  it('a cleared sale price drops the sale row; the regular price sells', () => {
    const live = liveMarginInsight(saved, { price: '10', salePrice: ' ', ...savedPrices })
    expect(live.unsaved).toBe(true)
    expect(live.insight.profit_at_sale_price_minor_units).toBeNull()
    expect(live.insight.margin_at_sale_price_basis_points).toBeNull()
    expect(live.insight.markup_at_sale_price_basis_points).toBeNull()
    expect(live.insight.total_profit_potential_minor_units).toBe(1500)
  })

  it('a half-typed or empty price keeps the saved figures', () => {
    for (const price of ['', '.', '-3', 'abc']) {
      const live = liveMarginInsight(saved, { price, salePrice: 'x', ...savedPrices })
      expect(live, price).toEqual({ insight: saved, unsaved: false })
    }
  })

  it('with no cost price there is nothing to recompute', () => {
    const noCost: SKUMarginInsight = { ...saved, cost_price_available: false, cost_price_minor_units: 0 }
    expect(liveMarginInsight(noCost, { price: '14', salePrice: '', ...savedPrices })).toEqual({ insight: noCost, unsaved: false })
  })
})
