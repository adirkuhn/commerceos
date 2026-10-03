import i18next from 'i18next'
import type { PurchaseOrderStatus, PurchaseOrderLineItem, ReceivingRecord, DiscrepancyType, LineItemType } from './types'
import type { SKU } from '../pms/types'

export const PROCUREMENT_PATH = '/procurement'

export const DISCREPANCIES_SEGMENT = 'discrepancies'

// Same list as Products.tsx's SKU currency select, for consistency.
export const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

export function todayDateString(): string {
  return new Date().toISOString().slice(0, 10)
}

export const STATUS_STYLE: Record<PurchaseOrderStatus, string> = {
  draft: 'bg-gray-500/15 text-ink-4',
  sent: 'bg-accent/15 text-accent',
  partially_received: 'bg-amber-500/15 text-warning',
  received: 'bg-green-500/15 text-success',
  cancelled: 'bg-red-500/15 text-danger',
  needs_review: 'bg-amber-500/20 text-warning',
}

// '' and 'product' both mean "an actual item" -- everything else is a
// charge/adjustment row that must never be forced through SKU matching.
export const LINE_TYPES: LineItemType[] = ['product', 'shipping', 'discount', 'credit_note', 'fee', 'other_charge']

export function lineTypeLabel(type: LineItemType): string {
  switch (type) {
    case 'product': return i18next.t('procurement:lineType.product')
    case 'shipping': return i18next.t('procurement:lineType.shipping')
    case 'discount': return i18next.t('procurement:lineType.discount')
    case 'credit_note': return i18next.t('procurement:lineType.credit_note')
    case 'fee': return i18next.t('procurement:lineType.fee')
    case 'other_charge': return i18next.t('procurement:lineType.other_charge')
    default: return type
  }
}

export function isChargeLineType(type: LineItemType): boolean {
  return type !== 'product'
}

// isReductionLineType mirrors backend LineItemType.IsReduction --
// unit_cost_minor_units is always entered as a non-negative amount (the
// "Unit cost" field has min="0"), so a discount/credit_note row's amount
// must be subtracted from the total here rather than relying on the stored
// value ever being negative.
function isReductionLineType(type: LineItemType): boolean {
  return type === 'discount' || type === 'credit_note'
}

export function signedLineItemAmount(li: Pick<PurchaseOrderLineItem, 'line_type' | 'quantity' | 'unit_cost_minor_units'>): number {
  const amount = li.unit_cost_minor_units * li.quantity
  return isReductionLineType(li.line_type) ? -amount : amount
}

// Backend's exact wording for procurementDomain.ErrAIParsingNotConfigured
// (application/procurement/extract_line_items_from_document.go) -- swapped
// for a friendlier message pointing at the manual-entry fallback that's
// already available right below (the "+ Add Line Item" button never
// depends on a successful parse).
const AI_NOT_CONFIGURED_MESSAGE = 'AI document parsing is not configured'

export function friendlyParsingError(errorMessage: string): string {
  if (errorMessage === AI_NOT_CONFIGURED_MESSAGE) {
    return i18next.t('procurement:parsing.aiNotConfigured')
  }
  return errorMessage || i18next.t('procurement:parsing.failed')
}

export const RECEIVING_STATUS_STYLE: Record<ReceivingRecord['status'], string> = {
  pending_review: 'bg-amber-500/20 text-warning',
  applied: 'bg-green-500/15 text-success',
}

export const DISCREPANCY_TYPES: DiscrepancyType[] = ['quantity_mismatch', 'missing_item', 'unexpected_item', 'expiry_mismatch', 'damaged_item']

export function discrepancyLabel(type: DiscrepancyType): string {
  switch (type) {
    case 'quantity_mismatch': return i18next.t('procurement:discrepancy.quantity_mismatch')
    case 'missing_item': return i18next.t('procurement:discrepancy.missing_item')
    case 'unexpected_item': return i18next.t('procurement:discrepancy.unexpected_item')
    case 'expiry_mismatch': return i18next.t('procurement:discrepancy.expiry_mismatch')
    case 'damaged_item': return i18next.t('procurement:discrepancy.damaged_item')
    default: return type
  }
}

/** A purchase order status as shown in badges (lower case; CSS capitalises). */
export function poStatusLabel(status: PurchaseOrderStatus): string {
  switch (status) {
    case 'draft': return i18next.t('procurement:poStatus.draft')
    case 'sent': return i18next.t('procurement:poStatus.sent')
    case 'partially_received': return i18next.t('procurement:poStatus.partially_received')
    case 'received': return i18next.t('procurement:poStatus.received')
    case 'cancelled': return i18next.t('procurement:poStatus.cancelled')
    case 'needs_review': return i18next.t('procurement:poStatus.needs_review')
    default: return status
  }
}

export function receivingStatusLabel(status: ReceivingRecord['status']): string {
  switch (status) {
    case 'pending_review': return i18next.t('procurement:receivingStatus.pending_review')
    case 'applied': return i18next.t('procurement:receivingStatus.applied')
    default: return status
  }
}

// The "Units per purchased item" to use once a line is linked to sku. A
// weight pool (unit_type kilo) keeps stock in grams, while suppliers
// invoice it per kg -- so its default becomes 1000 (book "50" = 50 kg).
// A value someone already changed from the default 1 is kept (e.g. a 5 kg
// bag = 5000).
export function unitsPerPurchasedItemOnLink(sku: Pick<SKU, 'unit_type'>, current: string): string {
  if (sku.unit_type === 'kilo' && (current === '' || current === '1')) return '1000'
  return current
}
