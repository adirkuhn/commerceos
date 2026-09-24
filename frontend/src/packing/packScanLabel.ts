import type { PackScan } from '../scanner/types'

const ACTION_LABEL: Record<PackScan['action_type'], string> = {
  pack: 'Packed', substitute: 'Substituted', freebie: 'Freebie', new_item: 'New item', refund: 'Refunded',
}

// One line of a packing session's log: what happened, the extra item's name
// (freebie/new item), and the quantity.
export function packScanLabel(s: PackScan): string {
  const name = (s.action_type === 'freebie' || s.action_type === 'new_item') && s.reason ? ` · ${s.reason}` : ''
  return `${ACTION_LABEL[s.action_type]}${name} · qty ${s.quantity}`
}
