import type { SKU } from '../pms/types'
import { formatSKUQuantity } from './inventoryFormat'

type StockSKU = Pick<SKU, 'id' | 'unit_type'>

// A product's total stock as people read it. A weight pool's ledger is in
// grams and every other SKU counts pieces, so the two are totalled apart:
// "6.00kg", "12", or "1.50kg · 5" for a pool sold in unit packages.
export function formatProductStock(skus: StockSKU[], stockBySKU: Record<string, number>): string {
  const sum = (list: StockSKU[]) => list.reduce((total, sku) => total + (stockBySKU[sku.id] ?? 0), 0)
  const pools = skus.filter(s => s.unit_type === 'kilo')
  const pieces = skus.filter(s => s.unit_type !== 'kilo')
  const parts: string[] = []
  if (pools.length > 0) parts.push(formatSKUQuantity({ unit_type: 'kilo' }, sum(pools)))
  if (pieces.length > 0 || pools.length === 0) parts.push(String(sum(pieces)))
  return parts.join(' · ')
}
