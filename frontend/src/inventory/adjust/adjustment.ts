import type { MovementType, RecordStockMovementInput } from '../types'
import type { StockUnit } from '../stockUnit'

// The rules behind "adjust stock": a person says which way stock moves
// (add / remove), how much, and why. Each reason maps to the movement type
// the backend expects for that sign (receipt > 0, sale < 0, adjustment
// either way). Damaged and lost goods are deliberately absent: they go
// through the write-off flow, which also keeps a discard record.
export type AdjustDirection = 'add' | 'remove'

export type AdjustReason = 'received' | 'returned' | 'found' | 'sold' | 'sample' | 'correction'

export const ADJUST_REASONS: Record<AdjustReason, { type: MovementType; directions: readonly AdjustDirection[] }> = {
  received: { type: 'receipt', directions: ['add'] },
  returned: { type: 'receipt', directions: ['add'] },
  found: { type: 'adjustment', directions: ['add'] },
  sold: { type: 'sale', directions: ['remove'] },
  sample: { type: 'adjustment', directions: ['remove'] },
  correction: { type: 'adjustment', directions: ['add', 'remove'] },
}

const REASON_ORDER: readonly AdjustReason[] = ['received', 'returned', 'found', 'sold', 'sample', 'correction']

export function reasonsFor(direction: AdjustDirection): AdjustReason[] {
  return REASON_ORDER.filter(r => ADJUST_REASONS[r].directions.includes(direction))
}

// The signed stored quantity for what was typed, or null unless it is a
// positive number of the SKU's unit (the direction carries the sign).
function signedDelta(direction: AdjustDirection, quantity: string, unit: StockUnit): number | null {
  const n = unit.parse(quantity)
  if (n === null || n <= 0) return null
  return direction === 'add' ? n : -n
}

export interface AdjustmentPreview {
  before: number
  after: number
  delta: number
  // Removing more than is on hand: the backend refuses it.
  short: boolean
}

export function adjustmentPreview({ onHand, direction, quantity, unit }: {
  onHand: number; direction: AdjustDirection; quantity: string; unit: StockUnit
}): AdjustmentPreview | null {
  const delta = signedDelta(direction, quantity, unit)
  if (delta === null) return null
  const after = onHand + delta
  return { before: onHand, after, delta, short: after < 0 }
}

export type AdjustmentError = 'quantity' | 'reason' | 'warehouse'

export type AdjustmentResult =
  | { ok: true; input: RecordStockMovementInput }
  | { ok: false; error: AdjustmentError }

// The movement to record, or which field is wrong. The stored reason is the
// reason's label as the person saw it, plus their note.
export function buildAdjustment({ warehouseId, batchId, direction, quantity, unit, reason, reasonLabel, note }: {
  warehouseId: string
  // '' is the warehouse's general ledger (stock without a lot).
  batchId: string
  direction: AdjustDirection
  quantity: string
  unit: StockUnit
  reason: AdjustReason
  reasonLabel: string
  note: string
}): AdjustmentResult {
  if (!warehouseId) return { ok: false, error: 'warehouse' }
  const delta = signedDelta(direction, quantity, unit)
  if (delta === null) return { ok: false, error: 'quantity' }
  if (!ADJUST_REASONS[reason].directions.includes(direction)) return { ok: false, error: 'reason' }
  const trimmed = note.trim()
  return {
    ok: true,
    input: {
      warehouse_id: warehouseId,
      batch_id: batchId || null,
      type: ADJUST_REASONS[reason].type,
      quantity_delta: delta,
      reason: trimmed ? `${reasonLabel} — ${trimmed}` : reasonLabel,
    },
  }
}
