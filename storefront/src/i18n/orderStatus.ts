import type { TFunction } from 'i18next'

/** An order status from the API, in the store's language (unknown ones as sent). */
export function orderStatusLabel(t: TFunction, status: string): string {
  const labels: Record<string, string> = {
    pending: t('orderStatus.pending'),
    processing: t('orderStatus.processing'),
    'on-hold': t('orderStatus.on-hold'),
    shipped: t('orderStatus.shipped'),
    completed: t('orderStatus.completed'),
    cancelled: t('orderStatus.cancelled'),
    refunded: t('orderStatus.refunded'),
    partially_refunded: t('orderStatus.partially_refunded'),
    failed: t('orderStatus.failed'),
  }
  return labels[status] ?? status
}
