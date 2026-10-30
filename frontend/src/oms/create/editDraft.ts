import type { PriceListRow } from '../../pricelist/types'
import type { Address, EditSalesOrderInput, SalesOrder, SalesOrderDetail, SalesOrderLineItem, SalesOrderStatus } from '../types'
import { toCreateInput, type DraftLine, type OrderDraft } from './orderDraft'

// Editing an existing order in the new-order form: the backend's
// SalesOrder.Editable rule, the order turned back into a draft, and the
// draft sent as PUT /sales-orders/{id}.

const EDITABLE_STATUSES: SalesOrderStatus[] = ['pending', 'processing', 'on-hold']
// Statuses whose stock the order already took (backend ReducesStock),
// before shipping.
const HOLDS_STOCK: SalesOrderStatus[] = ['processing', 'on-hold']

/** A manual order can be edited until it is fulfilled or shipped. */
export function canEditOrder(o: SalesOrder): boolean {
  return o.source === 'manual' && !o.fulfilled && EDITABLE_STATUSES.includes(o.status)
}

/** datetime-local input value for an ISO time, in the browser's zone. */
export function toLocalInput(iso: string): string {
  const d = new Date(iso)
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
  return d.toISOString().slice(0, 16)
}

// The rate a line was sold at, from its own totals -- for a product the
// price list no longer has.
function soldRate(li: SalesOrderLineItem): number {
  const tax = li.total_tax_minor_units
  const net = li.total_includes_tax ? li.total_minor_units - tax : li.total_minor_units
  return net > 0 && tax > 0 ? Math.round((tax * 10000) / net) : 0
}

const sameAddress = (a?: Address, b?: Address) => JSON.stringify(a ?? {}) === JSON.stringify(b ?? {})

/**
 * The order as the form's draft. Each line takes its product's current VAT
 * rate and stock from the price list (rows by SKU id); stock the order
 * itself holds counts as available to it, since editing gives it back.
 */
export function draftFromOrder(detail: SalesOrderDetail, rows: Map<string, PriceListRow>): OrderDraft {
  const o = detail.sales_order
  const held = HOLDS_STOCK.includes(o.status)
  const lines: DraftLine[] = detail.line_items.filter(li => li.sku_id).map(li => {
    const row = rows.get(li.sku_id!)
    return {
      key: `line-${li.id}`, skuId: li.sku_id!, label: li.name, skuCode: li.sku_code, quantity: li.quantity, unitPrice: li.price_minor_units,
      taxRateBasisPoints: row ? row.tax_rate_basis_points : soldRate(li),
      vatMissing: row?.tax_rate_configured === false,
      available: (row?.available_quantity ?? 0) + (held ? li.quantity : 0),
    }
  })
  const billing = o.billing ?? {}
  const same = sameAddress(o.shipping, billing) || !Object.values(o.shipping ?? {}).some(Boolean)
  return {
    warehouseId: o.warehouse_id ?? '', status: o.status, occurredAt: toLocalInput(o.occurred_at), number: o.number ?? '',
    note: o.customer_note ?? '', customerId: o.customer_id ?? null, billing, shipping: same ? {} : (o.shipping ?? {}),
    shippingSameAsBilling: same, lines, shipping_cost: o.shipping_total_minor_units ?? 0,
    feeLabel: o.fee_label ?? '', fee: o.fee_total_minor_units ?? 0,
  }
}

/** The draft as PUT /sales-orders/{id} takes it -- status and currency stay as they are. */
export function toEditInput(d: OrderDraft, ctx: { currency: string; defaultRateBasisPoints: number; pricesIncludeTax: boolean }): EditSalesOrderInput {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { status, currency_code, ...rest } = toCreateInput(d, ctx)
  return { ...rest, fee_label: d.feeLabel.trim(), fee_total_minor_units: d.fee }
}
