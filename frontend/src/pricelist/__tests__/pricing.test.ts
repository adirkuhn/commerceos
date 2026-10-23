import { describe, it, expect } from 'vitest'
import { applyDiscount, parseDiscount, listPrice, unitEconomics, quoteTotals } from '../pricing'
import type { PriceListRow } from '../types'

const row = (over: Partial<PriceListRow> = {}): PriceListRow => ({
  product_id: 'p-1', product_name: 'Feijão', variant_name: '', sku_id: 's-1', sku_code: 'FEI-1', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 1210, sale_price_minor_units: null, cost_price_minor_units: 500,
  available_quantity: 10, earliest_expiry: null, tax_rate_basis_points: 2100, target_markup_basis_points: 5000,
  ...over,
})

describe('parseDiscount', () => {
  it('reads a percentage from the URL, clamped to 0..100', () => {
    expect(parseDiscount('30')).toBe(30)
    expect(parseDiscount('12.5')).toBe(12.5)
    expect(parseDiscount('150')).toBe(100)
    expect(parseDiscount('-5')).toBe(0)
    expect(parseDiscount('abc')).toBe(0)
    expect(parseDiscount(null)).toBe(0)
  })
})

describe('applyDiscount', () => {
  it('takes the percentage off, rounded to the cent', () => {
    expect(applyDiscount(1000, 30)).toBe(700)
    expect(applyDiscount(999, 12.5)).toBe(874) // 874.125
    expect(applyDiscount(999, 0)).toBe(999)
    expect(applyDiscount(999, 100)).toBe(0)
  })
})

describe('listPrice', () => {
  it('is the sale price while a sale runs, else the regular price', () => {
    expect(listPrice(row())).toBe(1210)
    expect(listPrice(row({ sale_price_minor_units: 990 }))).toBe(990)
  })
})

describe('unitEconomics', () => {
  it('profit and markup exclude VAT when prices include it', () => {
    // 1210 incl. 21% = 1000 excl.; cost 500 -> profit 500, markup 100%.
    expect(unitEconomics(row(), 1210, true)).toEqual({ profit: 500, markupBasisPoints: 10000, belowTarget: false })
  })

  it('takes a VAT-exclusive price as it is', () => {
    expect(unitEconomics(row(), 1000, false)).toEqual({ profit: 500, markupBasisPoints: 10000, belowTarget: false })
  })

  it('flags a markup under the target', () => {
    // 847 incl. -> 700 excl.; profit 200 on cost 500 = 40% < 50% target.
    expect(unitEconomics(row(), 847, true)).toEqual({ profit: 200, markupBasisPoints: 4000, belowTarget: true })
  })

  it('flags selling below cost even without a target', () => {
    expect(unitEconomics(row({ target_markup_basis_points: 0 }), 400, false).belowTarget).toBe(true)
    expect(unitEconomics(row({ target_markup_basis_points: 0 }), 500, false).belowTarget).toBe(false)
  })

  it('knows nothing without a cost', () => {
    expect(unitEconomics(row({ cost_price_minor_units: null }), 1210, true)).toEqual({ profit: null, markupBasisPoints: null, belowTarget: false })
  })
})

describe('quoteTotals', () => {
  it('sums line totals and the profit of lines with a known cost', () => {
    const totals = quoteTotals([
      { row: row(), quantity: 10, unitPrice: 1210 },
      { row: row({ cost_price_minor_units: null }), quantity: 2, unitPrice: 300 },
    ], true)
    expect(totals).toEqual({ total: 12700, profit: 5000, profitComplete: false })
  })

  it('profit is complete when every line has a cost', () => {
    expect(quoteTotals([{ row: row(), quantity: 1, unitPrice: 1000 }], false)).toEqual({ total: 1000, profit: 500, profitComplete: true })
  })
})
