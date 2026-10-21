import type { TFunction } from 'i18next'
import type { CancelShipmentResult } from './api'

export interface CancelOutcome {
  deleted: boolean
  // MyParcel's own wording when it refused the delete.
  carrierError: string
  // The carrier can't delete a pre-announced label (PostNL): it stays,
  // free unless scanned, and drops off after 7 days. Not a refusal.
  labelKept: boolean
}

// cancelOutcomeText is a shipment cancellation's outcome in the reader's
// language. Without an outcome (shipments cancelled before outcomes were
// stored) it shows the stored English note.
export function cancelOutcomeText(t: TFunction<'shipping'>, outcome: CancelOutcome | null, fallback: string): string {
  if (!outcome) return fallback
  if (outcome.deleted) return t('cancelOutcome.deleted')
  if (outcome.labelKept) return t('cancelOutcome.labelKept')
  return t('cancelOutcome.notDeleted', { error: outcome.carrierError })
}

// outcomeOfCancelResult reads the cancel call's own result.
export function outcomeOfCancelResult(r: CancelShipmentResult): CancelOutcome {
  return { deleted: r.deleted_on_myparcel, carrierError: r.carrier_error ?? '', labelKept: r.label_kept ?? false }
}

// outcomeOfShipment reads a listed shipment's stored cancel outcome.
export function outcomeOfShipment(s: { cancel_deleted_on_carrier?: boolean | null; cancel_carrier_error?: string; cancel_label_kept?: boolean }): CancelOutcome | null {
  if (s.cancel_label_kept) return { deleted: false, carrierError: '', labelKept: true }
  if (s.cancel_deleted_on_carrier === undefined || s.cancel_deleted_on_carrier === null) return null
  return { deleted: s.cancel_deleted_on_carrier, carrierError: s.cancel_carrier_error ?? '', labelKept: false }
}
