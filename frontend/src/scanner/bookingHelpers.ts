import type { BookingDetail, BookingScan, MissingBookingItem } from './types'

// Batch picker value for a not-yet-real batch staged in this booking:
// PENDING_PREFIX + its lot number.
export const PENDING_PREFIX = 'pending:'

// Where a Remove or Damaged/lost row for this SKU most likely takes from:
// the batch its latest receive in this booking went into -- a pending lot,
// or an existing batch. General ledger ('') when nothing was received into
// either: booking stock never lands in the general ledger (a receive
// without a lot still makes its own batch), so defaulting there made the
// write-off fail at Finish. scans are this SKU's staged rows.
export function defaultTakeSource(scans: BookingScan[]): string {
  const receives = scans.filter(s => s.action_type === 'receive')
  const latest = receives.reduce<BookingScan | undefined>((acc, s) => (!acc || s.scanned_at >= acc.scanned_at ? s : acc), undefined)
  if (latest?.batch_id) return latest.batch_id
  if (latest?.lot_number) return `${PENDING_PREFIX}${latest.lot_number}`
  return ''
}

export interface PendingBatch {
  lotNumber: string
  expiryDate?: string | null
}

// A "receive" scan with a lot_number and no real batch_id is targeting a
// batch that won't exist until Finish (see CompleteBookingUseCase's replay,
// which resolves it by matching sku_id+lot_number across every staged row
// for the booking, regardless of who scanned it). Surfacing these in the
// batch picker is what lets a second scanner working the same booking add
// to a batch the first one just started, by name, instead of having to
// guess/retype the exact lot text.
export function pendingBatchesForSKU(detail: BookingDetail | null, skuId: string): PendingBatch[] {
  if (!detail) return []
  const seen = new Map<string, string | null | undefined>()
  for (const s of detail.scans) {
    if (s.sku_id !== skuId || s.action_type !== 'receive' || s.batch_id || !s.lot_number) continue
    if (!seen.has(s.lot_number)) seen.set(s.lot_number, s.expiry_date)
  }
  return Array.from(seen, ([lotNumber, expiryDate]) => ({ lotNumber, expiryDate }))
}

// A damaged/lost scan's quantity is always 0 (the write-off amount lives in
// write_off_quantity instead -- see BookingScan's doc comment), but it
// still counts toward this line's progress: it arrived, it just isn't
// sellable. Mirrors the backend's missingBookingItems fix exactly -- omitting
// write_off_quantity here made every damaged/lost unit look permanently
// missing, even once nothing actually was.
export function scannedQuantityBySKU(scans: BookingDetail['scans']): Map<string, number> {
  const bySKU = new Map<string, number>()
  for (const s of scans) {
    bySKU.set(s.sku_id, (bySKU.get(s.sku_id) ?? 0) + s.quantity + s.write_off_quantity)
  }
  return bySKU
}

// Mirrors PurchaseOrderLineItem.EffectiveUnitsPerPurchasedItem -- almost
// always 1, set above 1 when the supplier's line is a box/case that isn't
// itself a catalog SKU (see ScanBookingItemUseCase's scaleByLineItem).
export function effectiveUnitsPerPurchasedItem(li: { units_per_purchased_item?: number }): number {
  return li.units_per_purchased_item && li.units_per_purchased_item > 0 ? li.units_per_purchased_item : 1
}

// scans[].quantity is always stock units (ScanBookingItemUseCase scales a
// tap-to-book/RawSKU-matched entry up by EffectiveUnitsPerPurchasedItem
// before staging it -- see its own doc comment) -- this converts a SKU's
// summed stock-unit total back to purchased-item/box units, the same
// convention PurchaseOrderLineItem.Quantity itself uses, for display
// against it.
export function scannedPurchasedItems(li: { units_per_purchased_item?: number }, scannedStockQty: number): number {
  return scannedStockQty / effectiveUnitsPerPurchasedItem(li)
}

// Mirrors PurchaseOrderLineItem.EffectiveUnitsPerPurchasedItem +
// CompleteBookingUseCase's missing-items math exactly, computed from data
// already loaded client-side -- lets Finish show a preview before actually
// completing the booking, no extra backend call needed.
export function computeMissingItems(detail: BookingDetail): MissingBookingItem[] {
  const scannedBySKU = scannedQuantityBySKU(detail.scans)
  const missing: MissingBookingItem[] = []
  for (const li of detail.line_items) {
    const scannedQty = li.sku_id ? scannedBySKU.get(li.sku_id) ?? 0 : 0
    const expectedQty = li.quantity * effectiveUnitsPerPurchasedItem(li)
    if (scannedQty < expectedQty) {
      missing.push({ line_item_id: li.id, raw_description: li.raw_description, expected_qty: expectedQty, scanned_qty: scannedQty })
    }
  }
  return missing
}
