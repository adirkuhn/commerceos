import type { PickScan } from '../scanner/types'

export const ACTION_LABEL: Record<PickScan['action_type'], string> = {
  pick: 'Picked', substitute: 'Substituted', freebie: 'Freebie', new_item: 'New item', refund: 'Refunded',
}

export const PICKING_PATH = '/picking'
