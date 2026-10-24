import type { SalesOrderLineChange } from '../types'

export interface GroupedLineChanges {
  byLine: Map<string, SalesOrderLineChange[]>
  // Extras shipped without (no stock) -- they never became a line.
  notAdded: SalesOrderLineChange[]
}

export function groupLineChanges(changes: SalesOrderLineChange[]): GroupedLineChanges {
  const byLine = new Map<string, SalesOrderLineChange[]>()
  const notAdded: SalesOrderLineChange[] = []
  for (const c of changes) {
    if (!c.line_item_id) {
      notAdded.push(c)
      continue
    }
    byLine.set(c.line_item_id, [...(byLine.get(c.line_item_id) ?? []), c])
  }
  return { byLine, notAdded }
}

export function isExtraItem(c: Pick<SalesOrderLineChange, 'action'>): boolean {
  return c.action === 'freebie' || c.action === 'new_item'
}

// The product packed, by name; falls back to what staff typed and then the
// SKU code when the SKU was deleted.
export function changeProductLabel(c: Pick<SalesOrderLineChange, 'product_name' | 'variant_name' | 'reason' | 'sku_code'>): string {
  if (!c.product_name) return c.reason || c.sku_code
  return c.variant_name && c.variant_name !== c.product_name ? `${c.product_name} — ${c.variant_name}` : c.product_name
}
