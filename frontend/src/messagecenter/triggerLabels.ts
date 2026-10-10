import type { TFunction } from 'i18next'
import type { TriggerSubject } from './types'

// triggerLabels names every trigger subject for the admin.
export const triggerLabels = (t: TFunction<'messages'>): Record<TriggerSubject, string> => ({
  'pms.product.deleted': t('trigger.productDeleted'),
  'storefront.order.placed': t('trigger.storefrontOrderPlaced'),
  'pms.category.changed': t('trigger.categoryChanged'),
  'pms.product.changed': t('trigger.productChanged'),
  'payments.refund.manual_review_required': t('trigger.refundReview'),
  'payments.refund.issued': t('trigger.refundIssued'),
  'integrations.connection.connected': t('trigger.integrationConnected'),
  'oms.sales_order.status_changed': t('trigger.orderStatusChanged'),
  'inventory.stock.changed': t('trigger.stockChanged'),
  'shipping.shipment.fulfillment_signal': t('trigger.shipmentUpdate'),
  'inventory.stock.close_to_expiry_reported': t('trigger.stockCloseToExpiry'),
  'oms.sales_order.created': t('trigger.orderCreated'),
})

