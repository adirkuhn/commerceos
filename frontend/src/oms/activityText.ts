import type { TFunction } from 'i18next'
import { providerLabel } from '../webhooks/providerLabel'
import { SHIPPING_PROVIDERS } from '../shipping/shippingProviders'

// activityText is an order activity (timeline, a user's activity) in the
// reader's language, by its type (backend domain/oms ActivityType*).
// Unknown types show the stored English summary.
export function activityText(t: TFunction<'orders'>, a: { activity_type: string; summary: string; metadata?: Record<string, unknown> }): string {
  switch (a.activity_type) {
    case 'marked_picked': return t('activity.markedPicked')
    case 'myparcel_shipment_created': return t('activity.shipmentCreated', { provider: shippingProviderName(a.metadata) })
    case 'myparcel_shipment_failed': return t('activity.shipmentFailed', { provider: shippingProviderName(a.metadata) })
    case 'myparcel_shipment_cancelled': return t('activity.shipmentCancelled')
    case 'myparcel_order_exported': return t('activity.orderExported')
    case 'myparcel_order_export_failed': return t('activity.orderExportFailedLabelContinues')
    case 'myparcel_error': return t('activity.orderExportFailed')
    default: return a.summary
  }
}

// The provider a shipment activity was recorded for. Activities from
// before PostNL existed carry none -- those were all MyParcel.
function shippingProviderName(metadata: Record<string, unknown> | undefined): string {
  const provider = metadata?.provider
  return providerLabel(typeof provider === 'string' && provider ? provider : SHIPPING_PROVIDERS.myparcel)
}
