import type { TFunction } from 'i18next'

// activityText is an order activity (timeline, a user's activity) in the
// reader's language, by its type (backend domain/oms ActivityType*).
// Unknown types show the stored English summary.
export function activityText(t: TFunction<'orders'>, a: { activity_type: string; summary: string }): string {
  switch (a.activity_type) {
    case 'marked_picked': return t('activity.markedPicked')
    case 'myparcel_shipment_created': return t('activity.shipmentCreated')
    case 'myparcel_shipment_failed': return t('activity.shipmentFailed')
    case 'myparcel_shipment_cancelled': return t('activity.shipmentCancelled')
    case 'myparcel_order_exported': return t('activity.orderExported')
    case 'myparcel_order_export_failed': return t('activity.orderExportFailedLabelContinues')
    case 'myparcel_error': return t('activity.orderExportFailed')
    default: return a.summary
  }
}
