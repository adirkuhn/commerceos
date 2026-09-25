import type { PackingSessionDetail } from '../scanner/types'
import { packedQuantityByLineItem } from '../scanner/packingProgress'

export interface PackProgress {
  packedByLineItem: Map<string, number>
  totalItems: number
  packedItems: number
  // 0-100, for the progress bar width.
  percent: number
  allPacked: boolean
}

// How far along a packing session is: each line counts at most its ordered
// quantity, so over-covering one line can't make up for another.
export function packProgress(session: PackingSessionDetail | null): PackProgress {
  const packedByLineItem = session ? packedQuantityByLineItem(session.scans) : new Map<string, number>()
  const totalItems = session?.line_items.reduce((s, li) => s + li.quantity, 0) ?? 0
  const packedItems = session ? session.line_items.reduce((s, li) => s + Math.min(li.quantity, packedByLineItem.get(li.id) ?? 0), 0) : 0
  const percent = totalItems > 0 ? (packedItems / totalItems) * 100 : 0
  const allPacked = totalItems > 0 && packedItems >= totalItems
  return { packedByLineItem, totalItems, packedItems, percent, allPacked }
}
