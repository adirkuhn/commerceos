import { describe, it, expect } from 'vitest'
import { addLine, draftTotals, pricesIncludeTax, priceLine, toCreateInput, type DraftLine, type OrderDraft } from '../orderDraft'
import type { PriceListRow } from '../../../pricelist/types'

const row = (over: Partial<PriceListRow> = {}): PriceListRow => ({
  product_id: 'p-1', product_name: 'Feijão', variant_name: '', sku_id: 's-1', sku_code: 'FEI-1', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 1210, sale_price_minor_units: null, cost_price_minor_units: 500,
  available_quantity: 40, earliest_expiry: null, tax_rate_basis_points: 2100, target_markup_basis_points: 0, ...over,
})

describe('priceLine (same split as backend oms.PriceLine)', () => {
  it('takes VAT out of inclusive prices', () => {
    expect(priceLine(10, 1210, 2100, true)).toEqual({ total: 12100, tax: 2100 })
  })
  it('adds VAT on top of exclusive prices, rounded half up', () => {
    expect(priceLine(3, 999, 900, false)).toEqual({ total: 2997, tax: 270 })
  })
  it('no rate, no VAT', () => {
    expect(priceLine(1, 500, 0, true)).toEqual({ total: 500, tax: 0 })
  })
})

describe('pricesIncludeTax', () => {
  it('reads the setting, and a never-saved shop as tax-inclusive (the default)', () => {
    expect(pricesIncludeTax({ tenant_id: 't', prices_include_tax: false })).toBe(false)
    expect(pricesIncludeTax({ tenant_id: 't', prices_include_tax: true })).toBe(true)
    expect(pricesIncludeTax({ tenant_id: '', prices_include_tax: false })).toBe(true)
    expect(pricesIncludeTax(null)).toBe(true)
  })
})

describe('addLine', () => {
  it('adds a product at its sale price when one runs, else the regular price', () => {
    const lines = addLine([], row({ sale_price_minor_units: 990 }))
    expect(lines).toEqual([expect.objectContaining({ skuId: 's-1', label: 'Feijão', skuCode: 'FEI-1', quantity: 1, unitPrice: 990, taxRateBasisPoints: 2100, available: 40 })])
    expect(addLine([], row())[0].unitPrice).toBe(1210)
  })

  it('adding the same product again raises its quantity', () => {
    const once = addLine([], row())
    const twice = addLine(once, row())
    expect(twice).toHaveLength(1)
    expect(twice[0].quantity).toBe(2)
  })

  it('every line gets its own key, even after removals', () => {
    const a = addLine([], row())
    const b = addLine(a, row({ sku_id: 's-2' }))
    const removed = b.filter(l => l.skuId !== 's-1')
    const again = addLine(removed, row())
    expect(new Set(again.map(l => l.key)).size).toBe(again.length)
  })
})

const line = (over: Partial<DraftLine> = {}): DraftLine => ({
  key: 'k1', skuId: 's-1', label: 'Feijão', skuCode: 'FEI-1', quantity: 10, unitPrice: 1210, taxRateBasisPoints: 2100, available: 40, ...over,
})

describe('draftTotals', () => {
  it('inclusive prices: total is the sum of lines, shipping and fee; VAT is inside', () => {
    const t = draftTotals({ lines: [line(), line({ key: 'k2', skuId: 's-2', quantity: 2, unitPrice: 545, taxRateBasisPoints: 900 })],
      shipping: 695, fee: 2500, defaultRateBasisPoints: 2100, pricesIncludeTax: true })
    expect(t.subtotal).toBe(12100 + 1090)
    expect(t.lineTotals).toEqual([12100, 1090])
    // 2100 + 90 + shipping 695 incl. 21% (121) + fee 2500 incl. 21% (434)
    expect(t.tax).toBe(2100 + 90 + 121 + 434)
    expect(t.total).toBe(12100 + 1090 + 695 + 2500)
  })

  it('exclusive prices: VAT comes on top of everything', () => {
    const t = draftTotals({ lines: [line({ unitPrice: 1000 })], shipping: 500, fee: 0, defaultRateBasisPoints: 2100, pricesIncludeTax: false })
    expect(t.subtotal).toBe(10000)
    expect(t.tax).toBe(2100 + 105)
    expect(t.total).toBe(10000 + 500 + 2205)
  })
})

describe('toCreateInput', () => {
  it('sends lines with their split, totals, shipping, fee, customer and number', () => {
    const draft: OrderDraft = {
      warehouseId: 'wh-1', status: 'processing', occurredAt: '2026-10-24T10:00', number: ' B2B-1 ', note: 'Back door',
      customerId: 'c-1', billing: { first_name: 'Ana', city: 'Utrecht' }, shipping: {}, shippingSameAsBilling: true,
      lines: [line()], shipping_cost: 695, feeLabel: 'Pallet', fee: 2500,
    }
    const input = toCreateInput(draft, { currency: 'EUR', defaultRateBasisPoints: 2100, pricesIncludeTax: true })
    expect(input).toMatchObject({
      warehouse_id: 'wh-1', status: 'processing', number: 'B2B-1', customer_note: 'Back door', customer_id: 'c-1', currency_code: 'EUR',
      billing: { first_name: 'Ana', city: 'Utrecht' }, shipping: { first_name: 'Ana', city: 'Utrecht' },
      shipping_total_minor_units: 695, fee_label: 'Pallet', fee_total_minor_units: 2500,
      total_minor_units: 12100 + 695 + 2500, tax_total_minor_units: 2100 + 121 + 434,
      line_items: [{ sku_id: 's-1', name: 'Feijão', sku_code: 'FEI-1', quantity: 10, price_minor_units: 1210,
        subtotal_minor_units: 12100, subtotal_tax_minor_units: 2100, total_minor_units: 12100, total_tax_minor_units: 2100 }],
    })
    expect(input.occurred_at).toBe(new Date('2026-10-24T10:00').toISOString())
  })

  it('leaves out an empty fee and number', () => {
    const input = toCreateInput({
      warehouseId: 'wh-1', status: 'processing', occurredAt: '2026-10-24T10:00', number: '', note: '', customerId: null,
      billing: {}, shipping: { city: 'Delft' }, shippingSameAsBilling: false, lines: [line()], shipping_cost: 0, feeLabel: 'x', fee: 0,
    }, { currency: 'EUR', defaultRateBasisPoints: 2100, pricesIncludeTax: true })
    expect(input.fee_label).toBeUndefined()
    expect(input.fee_total_minor_units).toBeUndefined()
    expect(input.number).toBeUndefined()
    expect(input.customer_id).toBeUndefined()
    expect(input.shipping).toEqual({ city: 'Delft' })
  })
})
