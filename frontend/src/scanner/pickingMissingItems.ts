import type { PickScan, PickingSessionLineItem, MissingPickItem } from './types'

// A pick or substitute scan's net effect on a line's picked progress --
// mirrors the backend's own missingPickItems scoring exactly (freebie/
// new_item never count against an original line, they have no line_item_id).
// Shared by both picking UIs (scanner kiosk and desktop) so their divergence
// previews can never drift apart.
export function pickedQuantityByLineItem(scans: PickScan[]): Map<string, number> {
  const byLineItem = new Map<string, number>()
  for (const s of scans) {
    if (!s.line_item_id) continue
    if (s.action_type !== 'pick' && s.action_type !== 'substitute') continue
    byLineItem.set(s.line_item_id, (byLineItem.get(s.line_item_id) ?? 0) + s.quantity)
  }
  return byLineItem
}

// Mirrors CompletePickingUseCase's missingPickItems exactly, computed
// client-side from data already loaded -- lets a Finish/Complete action show
// a divergence preview before actually completing the session.
export function computeMissingItems(lineItems: PickingSessionLineItem[], scans: PickScan[]): MissingPickItem[] {
  const pickedByLineItem = pickedQuantityByLineItem(scans)
  const missing: MissingPickItem[] = []
  for (const li of lineItems) {
    const picked = pickedByLineItem.get(li.id) ?? 0
    if (picked < li.quantity) {
      missing.push({ line_item_id: li.id, name: li.name, expected_qty: li.quantity, picked_qty: picked })
    }
  }
  return missing
}
