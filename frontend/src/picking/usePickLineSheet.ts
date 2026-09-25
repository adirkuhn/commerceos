import { useCallback, useState } from 'react'
import type { PickingSessionLineItem } from '../scanner/types'

export interface PickLine {
  lineItem: PickingSessionLineItem
  pickedSoFar: number
}

export interface PickLineSheet {
  pickLine: PickLine | null
  pickQty: number
  setPickQty: (qty: number) => void
  open: (lineItem: PickingSessionLineItem, pickedSoFar: number) => void
  close: () => void
}

// The side sheet opened by tapping a line: pick a specific quantity of it.
// Prefills whatever is still left to pick (at least 1).
export function usePickLineSheet(): PickLineSheet {
  const [pickLine, setPickLine] = useState<PickLine | null>(null)
  const [pickQty, setPickQty] = useState(1)

  const open = useCallback((lineItem: PickingSessionLineItem, pickedSoFar: number) => {
    if (!lineItem.sku_id) return // an unmatched Woo-import line has no SKU to pick against -- must be resolved elsewhere first
    setPickLine({ lineItem, pickedSoFar })
    setPickQty(Math.max(1, lineItem.quantity - pickedSoFar))
  }, [])

  const close = useCallback(() => setPickLine(null), [])

  return { pickLine, pickQty, setPickQty, open, close }
}
