import type { SalesOrder, SalesOrderStatus, PaymentStatus } from './types'

// An order's journey as the coupon legs of a ticket: paid, picked, packed,
// shipped. Legs are not strictly sequential (an invoice order ships before
// it is paid), so each is done on its own facts and the first unfinished
// one is the current leg. A cancelled, failed or refunded order is void:
// its legs keep what happened, none is current.

export type LegId = 'paid' | 'picked' | 'packed' | 'shipped'
export type LegState = 'done' | 'current' | 'upcoming'

export interface Leg {
  id: LegId
  state: LegState
  // When the leg was done, where the order records it.
  at: string | null
}

export type LegOrder = Pick<SalesOrder, 'status' | 'payment_status' | 'fulfilled' | 'packed' | 'date_paid' | 'date_completed'>

const VOID_STATUSES: readonly SalesOrderStatus[] = ['cancelled', 'failed', 'refunded']
const SHIPPED_STATUSES: readonly SalesOrderStatus[] = ['shipped', 'completed']
// Money that was refunded was paid first.
const PAID_STATUSES: readonly PaymentStatus[] = ['paid', 'refunded', 'partially_refunded']

export function fulfilmentLegs(order: LegOrder): { voided: boolean; legs: Leg[] } {
  const shipped = SHIPPED_STATUSES.includes(order.status)
  // Picking is optional: a packed order counts as picked, and goods that
  // shipped were picked and packed even when that happened outside CommerceOS.
  const packed = !!order.packed || shipped
  const picked = !!order.fulfilled || packed
  const done: Record<LegId, boolean> = {
    paid: PAID_STATUSES.includes(order.payment_status),
    picked,
    packed,
    shipped,
  }
  const at: Record<LegId, string | null> = {
    paid: order.date_paid ?? null,
    picked: null,
    packed: null,
    shipped: order.date_completed ?? null,
  }
  const voided = VOID_STATUSES.includes(order.status)
  const ids: LegId[] = ['paid', 'picked', 'packed', 'shipped']
  const current = voided ? undefined : ids.find(id => !done[id])
  return {
    voided,
    legs: ids.map(id => ({ id, state: done[id] ? 'done' : id === current ? 'current' : 'upcoming', at: at[id] })),
  }
}
