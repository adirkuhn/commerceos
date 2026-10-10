import type { TFunction } from 'i18next'

export interface CancelOutcome {
  deleted: boolean
  // MyParcel's own wording when it refused the delete.
  carrierError: string
}

// cancelOutcomeText is a shipment cancellation's outcome in the reader's
// language. Without an outcome (shipments cancelled before outcomes were
// stored) it shows the stored English note.
export function cancelOutcomeText(t: TFunction<'shipping'>, outcome: CancelOutcome | null, fallback: string): string {
  if (!outcome) return fallback
  return outcome.deleted ? t('cancelOutcome.deleted') : t('cancelOutcome.notDeleted', { error: outcome.carrierError })
}

// outcomeOfShipment reads a listed shipment's stored cancel outcome.
export function outcomeOfShipment(s: { cancel_deleted_on_carrier?: boolean | null; cancel_carrier_error?: string }): CancelOutcome | null {
  if (s.cancel_deleted_on_carrier === undefined || s.cancel_deleted_on_carrier === null) return null
  return { deleted: s.cancel_deleted_on_carrier, carrierError: s.cancel_carrier_error ?? '' }
}
