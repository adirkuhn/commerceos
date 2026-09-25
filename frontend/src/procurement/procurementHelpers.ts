import type { PurchaseOrderStatus, PurchaseOrderLineItem, ReceivingRecord, DiscrepancyType, LineItemType } from './types'

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

export function formatMoney(minorUnits: number, currencyCode: string): string {
  return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
}

// '' and 'product' both mean "an actual item" -- everything else is a
// charge/adjustment row that must never be forced through SKU matching.
export const LINE_TYPE_LABELS: Record<Exclude<LineItemType, ''>, string> = {
  product: 'Product',
  shipping: 'Shipping',
  discount: 'Discount',
  credit_note: 'Credit note',
  fee: 'Fee',
  other_charge: 'Other charge',
}

export function isChargeLineType(type: LineItemType): boolean {
  return type !== '' && type !== 'product'
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
    return 'No AI provider is configured, so this document couldn’t auto-parse into line items. The document is saved below — add line items manually, or connect an AI provider in Settings → Integrations.'
  }
  return errorMessage || 'Failed to parse document'
}

export const RECEIVING_STATUS_STYLE: Record<ReceivingRecord['status'], string> = {
  pending_review: 'bg-amber-500/20 text-warning',
  applied: 'bg-green-500/15 text-success',
}

export const DISCREPANCY_LABEL: Record<DiscrepancyType, string> = {
  quantity_mismatch: 'Quantity mismatch',
  missing_item: 'Missing from invoice',
  unexpected_item: 'Not on purchase order',
  expiry_mismatch: 'Expiry mismatch',
  damaged_item: 'Damaged',
}

// ── Booking discrepancies report ────────────────────────────────────────────

export function formatDiscrepancyDate(iso?: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}
