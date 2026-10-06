import { useEffect, useState } from 'react'
import type { PlanOrderLine, RecommendationLine } from '../types'

const initial = (lines: RecommendationLine[]) =>
  Object.fromEntries(lines.map(l => [l.sku_id, String(l.purchase_quantity)]))

// The quantities about to be ordered, prefilled with the plan's and
// editable per line (as typed, so a cleared input stays empty). Reset when
// the plan reloads.
export function useOrderQuantities(lines: RecommendationLine[]) {
  const [typed, setTyped] = useState<Record<string, string>>(() => initial(lines))
  useEffect(() => { setTyped(initial(lines)) }, [lines])

  const quantity = (skuId: string): number => {
    const n = Number(typed[skuId])
    return Number.isInteger(n) && n > 0 ? n : 0
  }
  const orderLines: PlanOrderLine[] = lines
    .map(l => ({ sku_id: l.sku_id, quantity: quantity(l.sku_id) }))
    .filter(l => l.quantity > 0)

  return {
    typed: (skuId: string) => typed[skuId] ?? '',
    setTyped: (skuId: string, value: string) => setTyped(prev => ({ ...prev, [skuId]: value })),
    quantity,
    orderLines,
  }
}
