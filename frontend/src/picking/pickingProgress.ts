import type { PickingSessionDetail } from '../scanner/types'
import { pickedQuantityByLineItem } from '../scanner/pickingMissingItems'

export interface PickProgress {
  pickedByLineItem: Map<string, number>
  totalItems: number
  pickedItems: number
  // 0-100, for the progress bar width.
  percent: number
  allPicked: boolean
}

// How far along a picking session is. Over-picking a line never counts
// beyond its ordered quantity, so extras can't mask another line's gap.
export function pickProgress(session: PickingSessionDetail | null): PickProgress {
  const pickedByLineItem = session ? pickedQuantityByLineItem(session.scans) : new Map<string, number>()
  const totalItems = session?.line_items.reduce((s, li) => s + li.quantity, 0) ?? 0
  const pickedItems = session ? session.line_items.reduce((s, li) => s + Math.min(li.quantity, pickedByLineItem.get(li.id) ?? 0), 0) : 0
  const percent = totalItems > 0 ? (pickedItems / totalItems) * 100 : 0
  const allPicked = totalItems > 0 && pickedItems >= totalItems
  return { pickedByLineItem, totalItems, pickedItems, percent, allPicked }
}
