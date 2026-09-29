import i18next from 'i18next'
import type { PackScan } from '../scanner/types'

function actionLabel(action: PackScan['action_type']): string {
  switch (action) {
    case 'pack': return i18next.t('packing:action.pack')
    case 'substitute': return i18next.t('packing:action.substitute')
    case 'freebie': return i18next.t('packing:action.freebie')
    case 'new_item': return i18next.t('packing:action.new_item')
    case 'refund': return i18next.t('packing:action.refund')
    default: return action
  }
}

// One line of a packing session's log: what happened, the extra item's name
// (freebie/new item), and the quantity.
export function packScanLabel(s: PackScan): string {
  const name = (s.action_type === 'freebie' || s.action_type === 'new_item') && s.reason ? ` · ${s.reason}` : ''
  return i18next.t('packing:scanLog.entry', { action: actionLabel(s.action_type), name, quantity: s.quantity })
}
