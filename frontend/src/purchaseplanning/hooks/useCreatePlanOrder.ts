import { useCallback } from 'react'
import { PROCUREMENT_PATH } from '../../procurement/procurementHelpers'
import { useStackNavigation } from '../../routing/useStackNavigation'
import * as api from '../api'
import type { PlanOrderLine } from '../types'

// Creates the draft PO and opens it in the procurement editor (Back
// returns to the plan). Errors propagate for the card to show.
export function useCreatePlanOrder(token: string) {
  const { open } = useStackNavigation()
  return useCallback(async (supplierId: string, lines: PlanOrderLine[]) => {
    const { purchase_order_id: id } = await api.createPurchaseOrderFromPlan(token, { supplier_id: supplierId, lines })
    open(`${PROCUREMENT_PATH}/${encodeURIComponent(id)}`)
  }, [token, open])
}
