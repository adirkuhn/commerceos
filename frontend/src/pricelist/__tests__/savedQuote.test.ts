import { describe, it, expect } from 'vitest'
import { buildQuoteInput, isSameQuote, savedQuoteInput, savedQuoteParams, withSavedQuote, withoutQuote } from '../savedQuote'
import type { PriceListRow, SavedQuote } from '../types'

const row = (sku: string, price: number, sale: number | null = null): PriceListRow => ({
  product_id: `p-${sku}`, product_name: sku, variant_name: '', sku_id: sku, sku_code: sku, unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: price, sale_price_minor_units: sale, cost_price_minor_units: 100,
  available_quantity: 10, tax_rate_basis_points: 900, target_markup_basis_points: 0,
})

const saved: SavedQuote = {
  id: 'q-1', number: 7, reference: 'Q-0007', name: 'Sabor', currency_code: 'EUR', discount_basis_points: 1250,
  line_count: 2, total_minor_units: 0, created_at: '', updated_at: '',
  lines: [
    { product_id: 'p-a', sku_id: 'a', quantity: 10, unit_price_minor_units: 875, line_total_minor_units: 8750 },
    { product_id: 'p-b', sku_id: 'b', quantity: 0, unit_price_minor_units: 1200, line_total_minor_units: 0 },
  ],
}

describe('savedQuoteParams', () => {
  it('opens a saved quote with its stored prices and discount', () => {
    expect(savedQuoteParams(saved)).toEqual({ tab: 'quote', quote: 'q-1', discount: '12.5', lines: 'p-a.a.10.875,p-b.b.0.1200' })
  })

  it('leaves a zero discount and no lines out of the URL', () => {
    expect(savedQuoteParams({ ...saved, discount_basis_points: 0, lines: [] })).toEqual({ tab: 'quote', quote: 'q-1' })
  })
})

describe('buildQuoteInput', () => {
  it('stores each line at its shown price: typed, else the discounted (sale) price', () => {
    const rows = new Map([['a', row('a', 1000)], ['b', row('b', 2000, 1600)]])
    const input = buildQuoteInput('  Sabor ', 25, [
      { productId: 'p-a', skuId: 'a', quantity: 3, unitPrice: 999 },
      { productId: 'p-b', skuId: 'b', quantity: 2 },
    ], rows)

    expect(input).toEqual({
      name: 'Sabor', discount_basis_points: 2500, currency_code: 'EUR',
      lines: [
        { product_id: 'p-a', sku_id: 'a', quantity: 3, unit_price_minor_units: 999 },
        { product_id: 'p-b', sku_id: 'b', quantity: 2, unit_price_minor_units: 1200 },
      ],
    })
  })

  it('keeps a line whose product is gone only when its price is known', () => {
    const input = buildQuoteInput('x', 0, [
      { productId: 'p-gone', skuId: 'gone', quantity: 1, unitPrice: 500 },
      { productId: 'p-unknown', skuId: 'unknown', quantity: 1 },
    ], new Map(), 'EUR')

    expect(input.lines.map(l => l.sku_id)).toEqual(['gone'])
    expect(input.currency_code).toBe('EUR')
  })

  it('an empty quote has no currency', () => {
    expect(buildQuoteInput('x', 0, [], new Map()).currency_code).toBeUndefined()
  })
})

describe('isSameQuote', () => {
  it('is the saved quote until name, discount, a quantity or a price changes', () => {
    const asSaved = savedQuoteInput(saved)
    expect(isSameQuote(asSaved, savedQuoteInput(saved))).toBe(true)
    expect(isSameQuote({ ...asSaved, currency_code: undefined }, asSaved)).toBe(true)
    expect(isSameQuote({ ...asSaved, name: 'Other' }, asSaved)).toBe(false)
    expect(isSameQuote({ ...asSaved, discount_basis_points: 0 }, asSaved)).toBe(false)
    expect(isSameQuote({ ...asSaved, lines: [{ ...asSaved.lines[0], quantity: 11 }, asSaved.lines[1]] }, asSaved)).toBe(false)
  })
})

describe('withSavedQuote / withoutQuote', () => {
  it('switches the editor to a saved quote, keeping unrelated params', () => {
    const prev = new URLSearchParams('tab=saved&qs=sab&discount=30&lines=x.y.1&quote=old')
    expect(Object.fromEntries(withSavedQuote(prev, { ...saved, discount_basis_points: 0, lines: [] })))
      .toEqual({ tab: 'quote', qs: 'sab', quote: 'q-1' })
  })

  it('starts an empty quote', () => {
    const prev = new URLSearchParams('tab=quote&discount=30&lines=x.y.1&quote=old')
    expect(Object.fromEntries(withoutQuote(prev))).toEqual({ tab: 'quote' })
  })
})
