import i18next from 'i18next'
import type { SalesOrderStatus } from './types'

/** A sales order status in the admin language (unknown ones as sent). */
export function salesOrderStatusLabel(status: SalesOrderStatus | string): string {
  switch (status) {
    case 'pending': return i18next.t('orders:status.pending')
    case 'processing': return i18next.t('orders:status.processing')
    case 'on-hold': return i18next.t('orders:status.onHold')
    case 'shipped': return i18next.t('orders:status.shipped')
    case 'completed': return i18next.t('orders:status.completed')
    case 'cancelled': return i18next.t('orders:status.cancelled')
    case 'refunded': return i18next.t('orders:status.refunded')
    case 'failed': return i18next.t('orders:status.failed')
    case 'partially_refunded': return i18next.t('orders:status.partiallyRefunded')
    default: return status
  }
}
