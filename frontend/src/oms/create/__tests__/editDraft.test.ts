import { describe, it, expect } from 'vitest'
import { canEditOrder, draftFromOrder, toEditInput } from '../editDraft'
import type { PriceListRow } from '../../../pricelist/types'
import type { SalesOrder, SalesOrderDetail, SalesOrderLineItem } from '../../types'

const order = (over: Partial<SalesOrder> = {}): SalesOrder => ({
  id: 'so-1', source: 'manual', status: 'processing', number: 'CO-1001', warehouse_id: 'wh-1', customer_id: 'c-9',
  occurred_at: '2026-10-26T10:00:00Z', customer_note: 'Back door', currency_code: 'EUR',
  shipping_total_minor_units: 799, fee_label: 'Pallet', fee_total_minor_units: 2500, tax_total_minor_units: 0, total_minor_units: 8382,
  billing: { company: 'Claus B.V.', city: 'Hoofddorp' }, shipping: { company: 'Claus B.V.', city: 'Hoofddorp' },
  fulfilled: false, ...over,
} as SalesOrder)

const line = (over: Partial<SalesOrderLineItem> = {}): SalesOrderLineItem => ({
  id: 'li-1', sku_id: 'sku-1', name: 'Chá Mate', sku_code: 'MATE-100', barcode: '', quantity: 17, price_minor_units: 299,
  subtotal_minor_units: 5083, subtotal_tax_minor_units: 0, total_minor_units: 5083, total_tax_minor_units: 0, total_includes_tax: true, ...over,
})

const row = (over: Partial<PriceListRow> = {}): PriceListRow => ({
  product_id: 'p-1', product_name: 'Chá Mate', variant_name: '', sku_id: 'sku-1', sku_code: 'MATE-100', unit_type: 'unit', currency_code: 'EUR',
  price_minor_units: 299, available_quantity: 3, tax_rate_basis_points: 900, target_markup_basis_points: 0, ...over,
})

const detail = (o = order(), lines = [line()]): SalesOrderDetail => ({ sales_order: o, line_items: lines })

describe('canEditOrder', () => {
  it('allows a manual order before it is fulfilled or shipped', () => {
    for (const status of ['pending', 'processing', 'on-hold'] as const) expect(canEditOrder(order({ status }))).toBe(true)
  })

  it('refuses channel, fulfilled and closed orders', () => {
    expect(canEditOrder(order({ source: 'woocommerce' }))).toBe(false)
    expect(canEditOrder(order({ fulfilled: true }))).toBe(false)
    for (const status of ['shipped', 'completed', 'cancelled', 'refunded', 'failed'] as const) expect(canEditOrder(order({ status }))).toBe(false)
  })
})

describe('draftFromOrder', () => {
  it('fills the form from the order, with each line at its current VAT rate', () => {
    const d = draftFromOrder(detail(), new Map([['sku-1', row()]]))

    expect(d).toMatchObject({
      warehouseId: 'wh-1', number: 'CO-1001', note: 'Back door', customerId: 'c-9', status: 'processing',
      billing: { company: 'Claus B.V.', city: 'Hoofddorp' }, shippingSameAsBilling: true,
      shipping_cost: 799, feeLabel: 'Pallet', fee: 2500,
    })
    expect(d.lines).toHaveLength(1)
    expect(d.lines[0]).toMatchObject({ skuId: 'sku-1', label: 'Chá Mate', skuCode: 'MATE-100', quantity: 17, unitPrice: 299, taxRateBasisPoints: 900 })
  })

  it('counts the stock the order holds as available to it', () => {
    expect(draftFromOrder(detail(), new Map([['sku-1', row()]])).lines[0].available).toBe(20)
    expect(draftFromOrder(detail(order({ status: 'pending' })), new Map([['sku-1', row()]])).lines[0].available).toBe(3)
  })

  it('keeps the rate the line was sold at for a product no longer listed', () => {
    const d = draftFromOrder(detail(order(), [line({ total_minor_units: 1090, total_tax_minor_units: 90, quantity: 1, price_minor_units: 1090 })]), new Map())

    expect(d.lines[0].taxRateBasisPoints).toBe(900)
  })

  it('keeps a separate shipping address', () => {
    const d = draftFromOrder(detail(order({ shipping: { company: 'Depot', city: 'Delft' } })), new Map())

    expect(d.shippingSameAsBilling).toBe(false)
    expect(d.shipping).toEqual({ company: 'Depot', city: 'Delft' })
  })
})

describe('toEditInput', () => {
  it('sends the form without the create-only fields', () => {
    const d = draftFromOrder(detail(), new Map([['sku-1', row()]]))
    const input = toEditInput(d, { currency: 'EUR', defaultRateBasisPoints: 900, pricesIncludeTax: true })

    expect(input).not.toHaveProperty('status')
    expect(input).not.toHaveProperty('currency_code')
    expect(input).toMatchObject({ number: 'CO-1001', warehouse_id: 'wh-1', customer_note: 'Back door', fee_label: 'Pallet', fee_total_minor_units: 2500 })
    expect(input.line_items[0]).toMatchObject({ sku_id: 'sku-1', quantity: 17, total_minor_units: 5083, total_tax_minor_units: 420 })
  })
})
