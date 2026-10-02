import i18next from 'i18next'
import type { PaymentStatus } from './types'

// Mirrors the backend's domain/oms.PaymentStatus (derived there from the
// order's status and paid date -- never computed here).
export const PAYMENT_STATUSES: PaymentStatus[] = [
  'awaiting_payment', 'awaiting_confirmation', 'paid', 'failed', 'refunded', 'partially_refunded', 'cancelled_unpaid',
]

export type PaymentTone = 'success' | 'warning' | 'danger' | 'neutral'

export function paymentStatusTone(status: PaymentStatus): PaymentTone {
  switch (status) {
    case 'paid': return 'success'
    case 'awaiting_payment':
    case 'awaiting_confirmation': return 'warning'
    case 'failed': return 'danger'
    default: return 'neutral'
  }
}

/** A payment status in the admin language. */
export function paymentStatusLabel(status: PaymentStatus): string {
  return i18next.t(`orders:paymentStatus.${status}`)
}
