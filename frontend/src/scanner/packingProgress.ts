import type { PackScan, PickingSessionLineItem, MissingPackItem } from './types'

// How much of a line a pack row accounts for -- mirrors backend
// PackScan.Covers: packed and refunded units count as-is, a substitute
// counts the original quantity it replaces, extra items cover no line.
function covers(s: PackScan): number {
  switch (s.action_type) {
    case 'pack':
    case 'refund':
      return s.quantity
    case 'substitute':
      return s.original_quantity_covered ?? 0
    default:
      return 0
  }
}

export function packedQuantityByLineItem(scans: PackScan[]): Map<string, number> {
  const byLineItem = new Map<string, number>()
  for (const s of scans) {
    const qty = covers(s)
    if (!s.line_item_id || qty === 0) continue
    byLineItem.set(s.line_item_id, (byLineItem.get(s.line_item_id) ?? 0) + qty)
  }
  return byLineItem
}

// Mirrors CompletePackingUseCase's missingPackItems exactly, computed
// client-side so a Finish/Complete action can show a divergence preview
// before actually completing the session.
export function computeMissingPackItems(lineItems: PickingSessionLineItem[], scans: PackScan[]): MissingPackItem[] {
  const packedByLineItem = packedQuantityByLineItem(scans)
  const missing: MissingPackItem[] = []
  for (const li of lineItems) {
    const packed = packedByLineItem.get(li.id) ?? 0
    if (packed < li.quantity) {
      missing.push({ line_item_id: li.id, name: li.name, expected_qty: li.quantity, packed_qty: packed })
    }
  }
  return missing
}
