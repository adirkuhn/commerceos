import type { PackScan, PickingSessionLineItem, MissingPackItem } from './types'

// packedQuantityByLineItem mirrors pickedQuantityByLineItem -- simpler here
// since PackScan has no action_type/substitute-quantity split (see that
// type's doc comment).
export function packedQuantityByLineItem(scans: PackScan[]): Map<string, number> {
  const byLineItem = new Map<string, number>()
  for (const s of scans) {
    byLineItem.set(s.line_item_id, (byLineItem.get(s.line_item_id) ?? 0) + s.quantity)
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
