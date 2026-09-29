import i18next from 'i18next'
import type { PickScan } from '../scanner/types'

/** A scan's action in the admin's language. */
export function actionLabel(action: PickScan['action_type']): string {
  switch (action) {
    case 'pick': return i18next.t('picking:action.pick')
    case 'substitute': return i18next.t('picking:action.substitute')
    case 'freebie': return i18next.t('picking:action.freebie')
    case 'new_item': return i18next.t('picking:action.new_item')
    case 'refund': return i18next.t('picking:action.refund')
    default: return action
  }
}

export const PICKING_PATH = '/picking'
