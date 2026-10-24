import type { PriceListRow } from '../../pricelist/types'
import type { Address, CreateSalesOrderInput, SalesOrderStatus } from '../types'

// The admin's new-order form as plain data: lines, shipping cost, an extra
// fee, and the totals the server stores as sent. The VAT split is the same
// as backend oms.PriceLine/OrderTotals (and quote -> order), so an order
// typed here and one converted from a quote add up the same way. Money is
// integer minor units.

export interface DraftLine {
  key: string
  skuId: string
  label: string
  skuCode: string
  quantity: number
  unitPrice: number
  taxRateBasisPoints: number
  available: number
}

export interface OrderDraft {
  warehouseId: string
  status: SalesOrderStatus
  // datetime-local value
  occurredAt: string
  number: string
  note: string
  customerId: string | null
  billing: Address
  shipping: Address
  shippingSameAsBilling: boolean
  lines: DraftLine[]
  shipping_cost: number
  feeLabel: string
  fee: number
}

/** A never-saved shop reads as all zero values, but its default is tax-inclusive prices. */
export function pricesIncludeTax(settings: { tenant_id: string; prices_include_tax: boolean } | null | undefined): boolean {
  if (!settings || !settings.tenant_id) return true
  return settings.prices_include_tax
}

const roundDiv = (a: number, b: number) => Math.floor((a * 2 + b) / (b * 2))

/** quantity x unit price and the VAT in it (inclusive) or on top of it (exclusive). */
export function priceLine(quantity: number, unitPrice: number, rateBasisPoints: number, inclusive: boolean): { total: number; tax: number } {
  const total = quantity * unitPrice
  if (rateBasisPoints <= 0 || total <= 0) return { total, tax: 0 }
  const tax = inclusive ? total - roundDiv(total * 10000, 10000 + rateBasisPoints) : roundDiv(total * rateBasisPoints, 10000)
  return { total, tax }
}

let keySeq = 0
const nextKey = () => `line-${++keySeq}`

/** Adds a product at its sale price while one runs; the same product again raises its quantity. */
export function addLine(lines: DraftLine[], row: PriceListRow): DraftLine[] {
  if (lines.some(l => l.skuId === row.sku_id)) {
    return lines.map(l => (l.skuId === row.sku_id ? { ...l, quantity: l.quantity + 1 } : l))
  }
  const label = row.variant_name && row.variant_name !== row.product_name ? `${row.product_name} — ${row.variant_name}` : row.product_name
  return [...lines, {
    key: nextKey(), skuId: row.sku_id, label, skuCode: row.sku_code, quantity: 1,
    unitPrice: row.sale_price_minor_units ?? row.price_minor_units, taxRateBasisPoints: row.tax_rate_basis_points,
    available: row.available_quantity,
  }]
}

export interface DraftTotals {
  lineTotals: number[]
  subtotal: number
  tax: number
  total: number
}

/** Shipping and the fee are taxed at the shop's default rate. */
export function draftTotals(d: { lines: DraftLine[]; shipping: number; fee: number; defaultRateBasisPoints: number; pricesIncludeTax: boolean }): DraftTotals {
  const lines = d.lines.map(l => priceLine(l.quantity, l.unitPrice, l.taxRateBasisPoints, d.pricesIncludeTax))
  const extras = [d.shipping, d.fee].map(a => priceLine(1, a, d.defaultRateBasisPoints, d.pricesIncludeTax))
  const subtotal = lines.reduce((s, l) => s + l.total, 0)
  const tax = [...lines, ...extras].reduce((s, l) => s + l.tax, 0)
  const gross = subtotal + d.shipping + d.fee
  return { lineTotals: lines.map(l => l.total), subtotal, tax, total: d.pricesIncludeTax ? gross : gross + tax }
}

export function toCreateInput(d: OrderDraft, ctx: { currency: string; defaultRateBasisPoints: number; pricesIncludeTax: boolean }): CreateSalesOrderInput {
  const totals = draftTotals({ lines: d.lines, shipping: d.shipping_cost, fee: d.fee, ...ctx })
  const input: CreateSalesOrderInput = {
    warehouse_id: d.warehouseId, status: d.status, currency_code: ctx.currency,
    occurred_at: new Date(d.occurredAt).toISOString(), customer_note: d.note,
    billing: d.billing, shipping: d.shippingSameAsBilling ? d.billing : d.shipping,
    shipping_total_minor_units: d.shipping_cost, total_minor_units: totals.total, tax_total_minor_units: totals.tax,
    line_items: d.lines.map(l => {
      const { total, tax } = priceLine(l.quantity, l.unitPrice, l.taxRateBasisPoints, ctx.pricesIncludeTax)
      return {
        sku_id: l.skuId, name: l.label, sku_code: l.skuCode, quantity: l.quantity, price_minor_units: l.unitPrice,
        subtotal_minor_units: total, subtotal_tax_minor_units: tax, total_minor_units: total, total_tax_minor_units: tax,
      }
    }),
  }
  if (d.number.trim()) input.number = d.number.trim()
  if (d.customerId) input.customer_id = d.customerId
  if (d.fee > 0) {
    input.fee_label = d.feeLabel.trim()
    input.fee_total_minor_units = d.fee
  }
  return input
}
